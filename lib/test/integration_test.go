package test

import (
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

var imageName string

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(wrapCompose(m))
}

func TestGetLabels(t *testing.T) {
	registerLabelledContainers()
	assert := assert.New(t)
	cl := docker_registry.NewClient()
	cl.BecomeFoolishlyTrusting()

	labels, err := cl.LabelsForImageName(imageName)

	assert.Nil(err)
	assert.Contains(labels, sous.DockerRepoLabel)
	resetSingularity()
}

func TestGetRunningDeploymentSet(t *testing.T) {
	assert := assert.New(t)

	registerLabelledContainers()

	deps, which := deploymentWithRepo(assert, "https://github.com/opentable/docker-grafana.git")
	assert.Equal(3, len(deps))

	if which < 0 {
		assert.FailNow("If deployment is nil, other tests will crash")
	}

	grafana := deps[which]
	assert.Equal(singularityURL, grafana.Cluster)
	assert.Regexp("^0\\.1", grafana.Resources["cpus"])    // XXX strings and floats...
	assert.Regexp("^100\\.", grafana.Resources["memory"]) // XXX strings and floats...
	assert.Equal("1", grafana.Resources["ports"])         // XXX strings and floats...
	assert.Equal(17, grafana.SourceVersion.Version.Patch)
	assert.Equal("91495f1b1630084e301241100ecf2e775f6b672c", grafana.SourceVersion.Version.Meta)
	assert.Equal(1, grafana.NumInstances)
	assert.Equal(sous.ManifestKindService, grafana.Kind)

	resetSingularity()
}

func TestResolve(t *testing.T) {
	assert := assert.New(t)

	clusterDefs := sous.Defs{
		Clusters: sous.Clusters{
			singularityURL: sous.Cluster{
				BaseURL: singularityURL,
			},
		},
	}
	repoOne := "https://github.com/opentable/one.git"
	repoTwo := "https://github.com/opentable/two.git"
	repoThree := "https://github.com/opentable/three.git"

	stateOneTwo := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"one": manifest("opentable/one", "test-one", repoOne, "1.1.1"),
			"two": manifest("opentable/two", "test-two", repoTwo, "1.1.1"),
		},
	}
	stateTwoThree := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"two":   manifest("opentable/two", "test-two", repoTwo, "1.1.1"),
			"three": manifest("opentable/three", "test-three", repoThree, "1.1.1"),
		},
	}

	// ****
	err := sous.Resolve(stateOneTwo)
	if err != nil {
		assert.Fail(err.Error())
	}
	// ****
	time.Sleep(1 * time.Second)

	deps, which := deploymentWithRepo(assert, repoOne)
	log.Print(deps)
	assert.NotEqual(which, -1, "opentable/one not successfully deployed")
	one := deps[which]
	assert.Equal(1, one.NumInstances)

	which = findRepo(deps, repoTwo)
	assert.NotEqual(-1, which, "opentable/two not successfully deployed")
	two := deps[which]
	assert.Equal(1, two.NumInstances)

	// ****
	err = sous.Resolve(stateTwoThree)
	if err != nil {
		assert.Fail(err.Error())
	}
	// ****

	deps, which = deploymentWithRepo(assert, repoTwo)
	assert.NotEqual(-1, which, "opentable/two no longer deployed after resolve")
	assert.Equal(1, deps[which].NumInstances)

	which = findRepo(deps, repoThree)
	assert.NotEqual(-1, which, "opentable/three not successfully deployed")
	assert.Equal(1, deps[which].NumInstances)

	which = findRepo(deps, repoOne)
	if which != -1 {
		assert.Equal(0, deps[which].NumInstances)
	}

	resetSingularity()
}

func deploymentWithRepo(assert *assert.Assertions, repo string) (sous.Deployments, int) {
	deps, err := sous.GetRunningDeploymentSet([]string{singularityURL})
	if assert.Nil(err) {
		return deps, findRepo(deps, repo)
	}
	return sous.Deployments{}, -1
}

func findRepo(deps sous.Deployments, repo string) int {
	for i := range deps {
		log.Printf("deps[%d] = %+v\n", i, deps[i].SourceVersion.RepoURL)
		log.Printf("repo = %+v\n", repo)
		if deps[i].SourceVersion.RepoURL == sous.RepoURL(repo) {
			return i
		}
	}
	return -1
}

func manifest(drepo, containerDir, sourceURL, version string) *sous.Manifest {
	sv := sous.SourceVersion{
		RepoURL:    sous.RepoURL(sourceURL),
		RepoOffset: sous.RepoOffset(""),
		Version:    semv.MustParse(version),
	}

	in := buildImageName(drepo, version)
	buildAndPushContainer(containerDir, in)

	sous.InsertImageRecord(sv, in, "")

	return &sous.Manifest{
		Source: sous.SourceLocation{
			RepoURL:    sous.RepoURL(sourceURL),
			RepoOffset: sous.RepoOffset(""),
		},
		Owners: []string{`xyz`},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			singularityURL: sous.PartialDeploySpec{
				DeployConfig: sous.DeployConfig{
					Resources:    sous.Resources{}, //map[string]string
					Args:         []string{},
					Env:          sous.Env{}, //map[s]s
					NumInstances: 1,
				},
				Version: semv.MustParse(version),
				//clusterName: "it",
			},
		},
	}
}

func registerLabelledContainers() {
	registerAndDeploy(ip, "hello-labels", "hello-labels", []int32{})
	registerAndDeploy(ip, "hello-server-labels", "hello-server-labels", []int32{80})
	registerAndDeploy(ip, "grafana-repo", "grafana-labels", []int32{})
	imageName = fmt.Sprintf("%s/%s:%s", registryName, "grafana-repo", "latest")
}