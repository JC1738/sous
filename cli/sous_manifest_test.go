package cli

import (
	"bytes"
	"testing"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/samsalisbury/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestGet(t *testing.T) {
	out := &bytes.Buffer{}
	smg := &SousManifestGet{
		TargetManifestID: graph.TargetManifestID{
			Source: sous.SourceLocation{
				Repo: project1.Repo,
			},
		},
		State:     makeTestState(),
		OutWriter: graph.OutWriter(out),
	}
	res := smg.Execute([]string{})
	assert.Equal(t, 0, res.ExitCode())

	assert.Regexp(t, "github", out.String())
}

func TestManifestSet(t *testing.T) {
	mid := sous.ManifestID{
		Source: sous.SourceLocation{
			Repo: project1.Repo,
		},
	}
	baseState := makeTestState()
	mani, present := baseState.Manifests.Get(mid)
	require.True(t, present)
	mani.Flavor = "vanilla"
	yml, err := yaml.Marshal(mani)
	require.NoError(t, err)
	in := bytes.NewBuffer(yml)

	state := makeTestState()

	dummyWriter := sous.DummyStateManager{State: state}
	writer := graph.StateWriter{StateWriter: &dummyWriter}
	sms := &SousManifestSet{
		TargetManifestID: graph.TargetManifestID(mid),
		State:            state,
		InReader:         graph.InReader(in),
		StateWriter:      writer,
	}

	assert.Equal(t, 0, dummyWriter.WriteCount)
	res := sms.Execute([]string{})
	assert.Equal(t, 0, res.ExitCode())
	assert.Equal(t, 1, dummyWriter.WriteCount)

	upManifest, present := state.Manifests.Get(mid)
	require.True(t, present)
	assert.Equal(t, upManifest.Flavor, "vanilla")
}

func TestManifestYAML(t *testing.T) {
	uripath := "certainly/i/am/healthy"

	manifest := &sous.Manifest{
		Source: sous.SourceLocation{Repo: "gh"},
		Owners: []string{"sam", "judson"},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			"ci": sous.DeploySpec{
				DeployConfig: sous.DeployConfig{
					Resources: sous.Resources{
						"cpus":   "0.1",
						"memory": "100",
						"ports":  "1",
					},
					Startup: sous.Startup{
						CheckReadyURIPath: &uripath,
					},
				},
			},
		},
	}

	yml, err := yaml.Marshal(manifest)
	require.NoError(t, err)
	assert.Regexp(t, "(?i).*checkready.*", string(yml))

	newM := sous.Manifest{}
	err = yaml.Unmarshal(yml, &newM)
	require.NoError(t, err)

	assert.Equal(t, *newM.Deployments["ci"].Startup.CheckReadyURIPath, uripath)
}
