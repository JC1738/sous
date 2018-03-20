package docker

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/nyarly/spies"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func inMemoryDB(name string) *sql.DB {
	db, err := GetDatabase(&DBConfig{"sqlite3_sous", InMemoryConnection(name)})
	if err != nil {
		panic(err)
	}
	return db
}

func BenchmarkFPSchema(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fingerPrintSchema(schema)
	}
}

func BenchmarkRecreateDB(b *testing.B) {
	exec.Command("rm", "-rf", "testdata").Run()
	os.MkdirAll("testdata", os.ModeDir|os.ModePerm)
	defer func() {
		b.StopTimer()
		exec.Command("rm", "-rf", "testdata").Run()
	}()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetDatabase(&DBConfig{"sqlite3_sous", "testdata/test.db"})
		if err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkCreateDB(b *testing.B) {
	exec.Command("rm", "-rf", "testdata").Run()
	os.MkdirAll("testdata", os.ModeDir|os.ModePerm)
	defer func() {
		b.StopTimer()
		exec.Command("rm", "-rf", "testdata").Run()
	}()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dbName := fmt.Sprintf("testdata/test%d.db", i)
		_, err := GetDatabase(&DBConfig{"sqlite3_sous", dbName})
		if err != nil {
			b.Log(err)
		}
	}
}

func BenchmarkCreateInMemoryDB(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dbName := fmt.Sprintf("inmemory%d.db", i)
		_, err := GetDatabase(&DBConfig{"sqlite3_sous", InMemoryConnection(dbName)})
		if err != nil {
			b.Log(err)
		}
	}
}

func TestHarvestGuessedRepo(t *testing.T) {
	assert := assert.New(t)

	dc := docker_registry.NewDummyClient()

	host := "docker.repo.io"
	nc, err := NewNameCache(host, dc, logging.SilentLogSet(), inMemoryDB("guessed_repo"))
	assert.NoError(err)

	sl := sous.SourceLocation{
		Repo: "https://github.com/opentable/wackadoo",
		Dir:  "nested/there",
	}

	dc.MatchMethod("GetImageMetadata", spies.AnyArgs, docker_registry.Metadata{}, errors.Errorf("no such MD"))
	dc.FeedTags([]string{"something", "the other"})
	nc.harvest(sl)

	assert.Len(dc.CallsTo("AllTags"), 1)
}

func TestRoundTrip(t *testing.T) {
	assert := assert.New(t)
	dc := docker_registry.NewDummyClient()

	host := "docker.repo.io"
	base := "ot/wackadoo"

	nc, err := NewNameCache(host, dc, logging.SilentLogSet(), inMemoryDB("roundtrip"))
	assert.NoError(err)

	sv := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", "1.2.3")

	in := base + ":version-1.2.3"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"
	err = nc.Insert(sv, in, digest, []sous.Quality{})
	assert.NoError(err)

	cn, err := nc.GetCanonicalName(in)
	if assert.NoError(err) {
		assert.Equal(in, cn)
	}
	nin, _, err := nc.getImageName(sv)
	if assert.NoError(err) {
		assert.Equal(in, nin)
	}

	newSV := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", "1.2.42")

	cn = base + "@" + digest
	dc.FeedMetadata(docker_registry.Metadata{
		Registry:      host,
		Labels:        Labels(newSV),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	sv, err = nc.GetSourceID(NewBuildArtifact(in, nil))
	if assert.Nil(err) {
		assert.Equal(newSV, sv)
	}

	ncn, err := nc.GetCanonicalName(host + "/" + in)
	if assert.Nil(err) {
		assert.Equal(host+"/"+cn, ncn)
	}
}

func TestCanonicalizesToConfiguredRegistry(t *testing.T) {
	assert := assert.New(t)
	dc := docker_registry.NewDummyClient()

	dockerPrimary := "docker.repo.io"
	dockerCache := "nearby-docker-cache.repo.io"
	base := "ot/wackadoo"

	nc, err := NewNameCache(dockerCache, dc, logging.SilentLogSet(), inMemoryDB("canonsucceeds"))
	assert.NoError(err)

	in := base + ":version-1.2.3"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"

	primaryTagName := dockerPrimary + "/" + in
	primaryDigestName := dockerCache + "/" + base + "@" + digest
	cacheDigestName := dockerCache + "/" + base + "@" + digest

	newSV := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", "1.2.42")

	cn := base + "@" + digest
	dc.AddMetadata(dockerPrimary+`.*`, docker_registry.Metadata{
		Registry:      dockerPrimary,
		Labels:        Labels(newSV),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	dc.AddMetadata(dockerCache+`.*`, docker_registry.Metadata{
		Registry:      dockerPrimary,
		Labels:        Labels(newSV),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	sv, err := nc.GetSourceID(NewBuildArtifact(primaryTagName, nil))
	if assert.Nil(err) {
		assert.Equal(newSV, sv)
	}

	art, err := nc.GetArtifact(sv)
	if assert.NoError(err) {
		assert.Equal(cacheDigestName, art.Name)
	}

	// once for primary, once to check mirror
	assert.Len(dc.CallsTo("GetImageMetadata"), 2)

	sv, err = nc.GetSourceID(NewBuildArtifact(primaryDigestName, nil))
	if assert.Nil(err) {
		assert.Equal(newSV, sv)
	}

	// because previous responses should be cached
	assert.Len(dc.CallsTo("GetImageMetadata"), 2)
}

func TestLeavesRegistryUnchangedWhenUnknown(t *testing.T) {
	assert := assert.New(t)
	dc := docker_registry.NewDummyClient()

	dockerPrimary := "docker.repo.io"
	dockerCache := "nearby-docker-cache.repo.io"
	base := "ot/wackadoo"

	nc, err := NewNameCache(dockerCache, dc, logging.SilentLogSet(), inMemoryDB("canonsucceeds"))
	assert.NoError(err)

	in := base + ":version-1.2.3"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"

	primaryTagName := dockerPrimary + "/" + in
	primaryDigestName := dockerPrimary + "/" + base + "@" + digest

	newSV := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", "1.2.42")

	cn := base + "@" + digest
	dc.AddMetadata(dockerPrimary+`.*`, docker_registry.Metadata{
		Registry:      dockerPrimary,
		Labels:        Labels(newSV),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	/*
		NOTE MISSING:
		dc.AddMetadata(dockerCache+`.*`, docker_registry.Metadata{
	*/

	dc.MatchMethod("GetImageMetadata", spies.AnyArgs, docker_registry.Metadata{}, errors.Errorf("no such MD"))
	sv, err := nc.GetSourceID(NewBuildArtifact(primaryTagName, nil))
	if assert.Nil(err) {
		assert.Equal(newSV, sv)
	}

	art, err := nc.GetArtifact(sv)
	if assert.NoError(err) {
		assert.Equal(primaryDigestName, art.Name)
	}
}

// I'm still exploring what the problem is here...
func TestHarvestAlso(t *testing.T) {
	assert := assert.New(t)

	dc := docker_registry.NewDummyClient()

	host := "docker.repo.io"
	base := "ot/wackadoo"
	repo := "github.com/opentable/test-app"

	nc, err := NewNameCache(host, dc, logging.SilentLogSet(), inMemoryDB("harvest_also"))
	assert.NoError(err)

	stuffBA := func(n, v string) sous.SourceID {
		ba := &sous.BuildArtifact{
			Name: n,
			Type: "docker",
		}

		sv := sous.MustNewSourceID(repo, "", v)

		in := base + ":version-" + v
		digBs := sha256.Sum256([]byte(in))
		digest := hex.EncodeToString(digBs[:])
		cn := base + "@sha256:" + digest

		dc.FeedMetadata(docker_registry.Metadata{
			Registry:      host,
			Labels:        Labels(sv),
			Etag:          digest,
			CanonicalName: cn,
			AllNames:      []string{cn, in},
		})
		sid, err := nc.GetSourceID(ba)
		assert.NoError(err)
		assert.NotNil(sid)
		return sid
	}
	sid1 := stuffBA("tom", "0.2.1")
	sid2 := stuffBA("dick", "0.2.2")
	sid3 := stuffBA("harry", "0.2.3")

	_, err = nc.GetArtifact(sid1) //which should not miss
	assert.NoError(err)
	_, err = nc.GetArtifact(sid2) //which should not miss
	assert.NoError(err)
	_, err = nc.GetArtifact(sid3) //which should not miss
	assert.NoError(err)
}

// This can happen e.g. if the same source gets built twice
func TestSecondCanonicalName(t *testing.T) {
	assert := assert.New(t)

	dc := docker_registry.NewDummyClient()

	host := "docker.repo.io"
	base := "ot/wackadoo"
	nc, err := NewNameCache(host, dc, logging.SilentLogSet(), inMemoryDB("secondCN"))
	require.NoError(t, err)

	repo := "github.com/opentable/test-app"

	stuffBA := func(digest string) sous.SourceID {
		n := "test-service"
		v := `0.1.2-ci1234`
		ba := &sous.BuildArtifact{
			Name: n,
			Type: "docker",
		}

		sv := sous.MustNewSourceID(repo, "", v)

		in := base + ":version-" + v
		cn := base + "@sha256:" + digest

		dc.FeedMetadata(docker_registry.Metadata{
			Registry:      host,
			Labels:        Labels(sv),
			Etag:          digest,
			CanonicalName: cn,
			AllNames:      []string{cn, in},
		})
		sid, err := nc.GetSourceID(ba)
		if !assert.NoError(err) {
			fmt.Println(err)
			nc.dump(os.Stderr)
		}
		assert.NotNil(sid)
		return sid
	}
	sid1 := stuffBA(`012345678901234567890123456789AB012345678901234567890123456789AB`)
	sid2 := stuffBA(`ABCDEFABCDEFABCDEABCDEABCDEABCDEABCDEABCDEABCDEABCDEF12341234566`)

	_, err = nc.GetArtifact(sid1) //which should not miss
	assert.NoError(err)

	_, err = nc.GetArtifact(sid2) //which should not miss
	assert.NoError(err)
}

func TestHarvesting(t *testing.T) {
	assert := assert.New(t)
	dc := docker_registry.NewDummyClient()

	host := "docker.repo.io"
	base := "wackadoo/nested/there"
	nc, err := NewNameCache(host, dc, logging.SilentLogSet(), inMemoryDB("harvesting"))
	assert.NoError(err)

	v := "1.2.3"
	sv := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", v)

	v2 := "2.3.4"
	sisterSV := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", v2)

	tag := "1.2.3"
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"
	cn := base + "@" + digest
	in := base + ":" + tag

	dc.AddMetadata(`.*`+in+`.*`, docker_registry.Metadata{
		Registry:      host,
		Labels:        Labels(sv),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	// a la a SetCollector getting the SV
	_, err = nc.GetSourceID(NewBuildArtifact(in, nil))
	assert.Nil(err)

	tag = "2.3.4"
	dc.FeedTags([]string{tag})
	digest = "sha256:abcdefabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeabcdeffffffff"
	cn = base + "@" + digest
	in = base + ":" + tag

	dc.AddMetadata(`.*`+in+`.*`, docker_registry.Metadata{
		Registry:      host,
		Labels:        Labels(sisterSV),
		Etag:          digest,
		CanonicalName: cn,
		AllNames:      []string{cn, in},
	})

	dc.MatchMethod("GetImageMetadata", spies.AnyArgs, docker_registry.Metadata{}, errors.Errorf("no such MD"))
	nin, err := nc.GetArtifact(sisterSV)
	if assert.NoError(err) {
		assert.Equal(host+"/"+cn, nin.Name)
	} else {
		t.Log(dc)
	}
}

func TestRecordAdvisories(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	dc := docker_registry.NewDummyClient()
	host := "docker.repo.io"
	base := "ot/wackadoo"
	nc, err := NewNameCache(host, dc, logging.SilentLogSet(), inMemoryDB("advisories"))
	require.NoError(err)
	v := "1.2.3"
	sv := sous.MustNewSourceID("https://github.com/opentable/wackadoo", "nested/there", v)
	digest := "sha256:012345678901234567890123456789AB012345678901234567890123456789AB"
	cn := base + "@" + digest

	qs := []sous.Quality{{Name: "ephemeral_tag", Kind: "advisory"}}

	err = nc.Insert(sv, cn, digest, qs)
	assert.NoError(err)

	arty, err := nc.GetArtifact(sv)
	assert.NoError(err)
	require.NotNil(arty)
	require.Len(arty.Qualities, 1)
	assert.Equal(arty.Qualities[0].Name, `ephemeral_tag`)
}

func TestDump(t *testing.T) {
	assert := assert.New(t)

	io := &bytes.Buffer{}

	dc := docker_registry.NewDummyClient()
	nc, err := NewNameCache("", dc, logging.SilentLogSet(), inMemoryDB("dump"))
	assert.NoError(err)

	nc.dump(io)
	assert.Regexp(`name_id`, io.String())
}

func TestMissingName(t *testing.T) {
	assert := assert.New(t)
	dc := docker_registry.NewDummyClient()
	nc, err := NewNameCache("", dc, logging.SilentLogSet(), inMemoryDB("missing"))
	assert.NoError(err)

	v := "4.5.6"
	sv := sous.MustNewSourceID("https://github.com/opentable/brand-new-idea", "nested/there", v)
	dc.AddMetadata(`.*opentable/brand-new-idea.*`, docker_registry.Metadata{})
	dc.MatchMethod("GetImageMetadata", spies.AnyArgs, docker_registry.Metadata{}, errors.Errorf("no such MD"))

	name, _, err := nc.getImageName(sv)
	assert.Equal("", name)
	assert.Error(err)
}

func TestUnion(t *testing.T) {
	assert := assert.New(t)

	left := []string{"a", "b", "c"}
	right := []string{"b", "c", "d"}

	all := union(left, right)
	assert.Equal(len(all), 4)
	assert.Contains(all, "a")
	assert.Contains(all, "b")
	assert.Contains(all, "c")
	assert.Contains(all, "d")
}
