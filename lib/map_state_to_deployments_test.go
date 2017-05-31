package sous

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

var project1 = SourceLocation{Repo: "github.com/user/project"}
var cluster1 = &Cluster{
	Name:    "cluster-1",
	Kind:    "singularity",
	BaseURL: "http://nothing.here.one",
	Env: EnvDefaults{
		"CLUSTER_LONG_NAME": Var("Cluster One"),
	},
}
var cluster2 = &Cluster{
	Name:    "cluster-2",
	Kind:    "singularity",
	BaseURL: "http://nothing.here.two",
	Env: EnvDefaults{
		"CLUSTER_LONG_NAME": Var("Cluster Two"),
	},
}

func makeTestDefs() Defs {
	return Defs{
		DockerRepo: "some.docker.repo",
		Clusters: Clusters{
			"cluster-1": cluster1,
			"cluster-2": cluster2,
		},
		EnvVars: EnvDefs{
			{
				Name:  "CLUSTER_LONG_NAME",
				Desc:  "The human-friendly name of this cluster.",
				Scope: "cluster",
				Type:  VarType("string"),
			},
		},
		Resources: FieldDefinitions{
			{Name: "cpus", Type: "float"},
			{Name: "mem", Type: "memory_size"},
		},
	}
}

var omalleyReadyURI = "/ow/baby"

func makeTestManifests() Manifests {
	return NewManifests(
		&Manifest{
			Source: project1,
			Owners: []string{"owner1"},
			Kind:   ManifestKindService,
			Deployments: DeploySpecs{
				"cluster-1": {
					Version: semv.MustParse("1.0.0"),
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus": "1",
							"mem":  "1024",
						},
						Env: Env{
							"ALL":               "IS ONE",
							"ENV_1":             "ENV ONE",
							"CLUSTER_LONG_NAME": "Cluster One",
						},
						Metadata: Metadata{
							"everybody": "wants to be a cat",
							"name":      "O'Malley",
						},
						NumInstances: 2,
						Startup: Startup{
							CheckReadyURIPath: &omalleyReadyURI,
						},
					},
				},
				"cluster-2": {
					Version: semv.MustParse("2.0.0"),
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus": "2",
							"mem":  "2048",
						},
						Env: Env{
							"ALL":               "IS ONE",
							"ENV_2":             "ENV TWO",
							"CLUSTER_LONG_NAME": "I Like To Call It 'Cluster Two'",
						},
						Metadata: Metadata{
							"everybody": "wants to be a cat",
							"name":      "Duchess",
						},
						NumInstances: 3,
					},
				},
			},
		},
		&Manifest{
			Source: project1,
			Flavor: "some-flavor",
			Owners: []string{"owner1flav"},
			Kind:   ManifestKindService,
			Deployments: DeploySpecs{
				"cluster-1": {
					Version: semv.MustParse("1.0.1"),
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus": "1.5",
							"mem":  "1024",
						},
						Env: Env{
							"ENV_1": "ENV ONE FLAVORED",
						},
						NumInstances: 4,
					},
				},
				"cluster-2": {
					Version: semv.MustParse("2.0.1"),
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus": "2.5",
							"mem":  "2048",
						},
						Env: Env{
							"ENV_2": "ENV TWO FLAVORED",
						},
						NumInstances: 5,
					},
				},
			},
		},
	)
}

func makeTestDeployments() (Deployments, error) {
	return makeTestManifests().Deployments(makeTestDefs())
}

var expectedDeployments = NewDeployments(
	&Deployment{
		SourceID:    project1.SourceID(semv.MustParse("1.0.0")),
		ClusterName: "cluster-1",
		Cluster:     cluster1,
		Kind:        ManifestKindService,
		Owners:      NewOwnerSet("owner1"),
		DeployConfig: DeployConfig{
			Resources: Resources{
				"cpus": "1",
				"mem":  "1024",
			},
			Env: Env{
				"ALL":               "IS ONE",
				"ENV_1":             "ENV ONE",
				"CLUSTER_LONG_NAME": "Cluster One",
			},
			Metadata: Metadata{
				"everybody": "wants to be a cat",
				"name":      "O'Malley",
			},
			NumInstances: 2,
			Startup: Startup{
				CheckReadyURIPath: &omalleyReadyURI,
			},
		},
	},
	&Deployment{
		SourceID:    project1.SourceID(semv.MustParse("2.0.0")),
		ClusterName: "cluster-2",
		Cluster:     cluster2,
		Kind:        ManifestKindService,
		Owners:      NewOwnerSet("owner1"),
		DeployConfig: DeployConfig{
			Resources: Resources{
				"cpus": "2",
				"mem":  "2048",
			},
			Env: Env{
				"ALL":   "IS ONE",
				"ENV_2": "ENV TWO",
				//"CLUSTER_LONG_NAME": "Cluster Two",
				"CLUSTER_LONG_NAME": "I Like To Call It 'Cluster Two'",
			},
			Metadata: Metadata{
				"everybody": "wants to be a cat",
				"name":      "Duchess",
			},
			NumInstances: 3,
		},
	},
	&Deployment{
		Flavor:      "some-flavor",
		SourceID:    project1.SourceID(semv.MustParse("1.0.1")),
		ClusterName: "cluster-1",
		Cluster:     cluster1,
		Kind:        ManifestKindService,
		Owners:      NewOwnerSet("owner1flav"),
		DeployConfig: DeployConfig{
			Resources: Resources{
				"cpus": "1.5",
				"mem":  "1024",
			},
			Env: Env{
				"ENV_1":             "ENV ONE FLAVORED",
				"CLUSTER_LONG_NAME": "Cluster One",
			},
			NumInstances: 4,
		},
	},
	&Deployment{
		Flavor:      "some-flavor",
		SourceID:    project1.SourceID(semv.MustParse("2.0.1")),
		ClusterName: "cluster-2",
		Cluster:     cluster2,
		Kind:        ManifestKindService,
		Owners:      NewOwnerSet("owner1flav"),
		DeployConfig: DeployConfig{
			Resources: Resources{
				"cpus": "2.5",
				"mem":  "2048",
			},
			Env: Env{
				"ENV_2":             "ENV TWO FLAVORED",
				"CLUSTER_LONG_NAME": "Cluster Two",
			},
			NumInstances: 5,
		},
	},
)

func TestState_DeploymentsCloned(t *testing.T) {
	actualDeployments, err := makeTestDeployments()
	if err != nil {
		t.Fatal(err)
	}
	actualDeployments = actualDeployments.Clone()

	exSnap := expectedDeployments.Snapshot()
	if len(actualDeployments.Snapshot()) != len(exSnap) {
		t.Error("deployments different lengths")
	}
	for id, expected := range exSnap {
		actual, ok := actualDeployments.Get(id)
		if !ok {
			t.Errorf("missing deployment %q", id)
			continue
		}
		if !actual.Equal(expected) {
			t.Errorf("%q\n\ngot:\n%v\n\nwant:\n%v\n", id, jsonDump(actual), jsonDump(expected))
		}
	}
}

var TestStateIndependentDeploySpecsState Manifests

func TestState_IndependentDeploySpecs(t *testing.T) {
	originalManifests := makeTestManifests()
	defs := makeTestDefs()
	originalDeployments, err := originalManifests.Deployments(defs)
	if err != nil {
		t.Fatal(err)
	}
	mid := ManifestID{Source: SourceLocation{Repo: "github.com/user/project"}}
	did := DeploymentID{ManifestID: mid, Cluster: "cluster-1"}
	originalDeployment, ok := originalDeployments.Get(did)
	if !ok {
		t.Fatalf("deployment %v not found", did)
	}
	originalJSON := jsonDump(originalDeployment)
	// We don't care about this result, but write it to global state to avoid
	// any compiler optimisation from eliding the call.
	TestStateIndependentDeploySpecsState, err = originalDeployments.PutbackManifests(defs, originalManifests)
	if err != nil {
		t.Error(err)
	}
	newDeployment, ok := originalDeployments.Get(did)
	if !ok {
		t.Fatalf("deployment %v went missing", did)
	}
	newJSON := jsonDump(newDeployment)

	if originalJSON != newJSON {
		t.Fatalf("original deployspec changed:\n\noriginal:\n%s\n\nnow:\n%s", originalJSON, newJSON)
	}
}

func TestState_Deployments(t *testing.T) {
	actualDeployments, err := makeTestDeployments()
	if err != nil {
		t.Fatal(err)
	}
	compareDeployments(t, expectedDeployments, actualDeployments)
}

func TestState_DeploymentsBounce(t *testing.T) {
	defs := makeTestDefs()
	bounceManifests, err := expectedDeployments.Clone().PutbackManifests(defs, makeTestManifests())
	if err != nil {
		t.Fatal(err)
	}
	actualDeployments, err := bounceManifests.Deployments(defs)
	if err != nil {
		t.Fatal(err)
	}
	compareDeployments(t, expectedDeployments, actualDeployments)
}

func TestDeployments_Manifests(t *testing.T) {
	defs := makeTestDefs()

	actualManifests, err := expectedDeployments.Clone().PutbackManifests(defs, makeTestManifests())
	if err != nil {
		t.Fatal(err)
	}
	expectedManifests := makeTestManifests()

	compareManifests(t, expectedManifests, actualManifests)
}

func TestDeployments_ManifestsBounce(t *testing.T) {
	defs := makeTestDefs()

	bounced := makeTestManifests()

	bounceDeployments, err := bounced.Deployments(makeTestDefs())
	if err != nil {
		t.Fatal(err)
	}

	actualManifests, err := bounceDeployments.PutbackManifests(defs, bounced)
	if err != nil {
		t.Fatal(err)
	}
	expectedManifests := makeTestManifests()

	compareManifests(t, expectedManifests, actualManifests)
}

func TestDeployments_PutbackManifestMissingManifestElides(t *testing.T) {
	defs := makeTestDefs()
	deps := NewDeployments(
		&Deployment{
			SourceID: project1.SourceID(semv.MustParse("2.0.0")),

			Cluster:     cluster1,
			ClusterName: cluster1.Name,
			DeployConfig: DeployConfig{
				Env: Env{
					"CLUSTER_LONG_NAME": "Cluster One",
					"PRESENT":           "here",
				},
			},
		},
	)

	zeroMs := NewManifests()
	ms, err := deps.PutbackManifests(defs, zeroMs)
	assert.NoError(t, err)
	m, yes := ms.Any(func(*Manifest) bool { return true })
	assert.True(t, yes)
	assert.NotContains(t, m.Deployments["cluster-1"].Env, "CLUSTER_LONG_NAME")
	assert.Contains(t, m.Deployments["cluster-1"].Env, "PRESENT")
}

func TestDeployments_PutbackManifestOriginalLacksClusterElides(t *testing.T) {
	defs := makeTestDefs()
	deps := NewDeployments(
		&Deployment{
			SourceID: project1.SourceID(semv.MustParse("2.0.0")),

			Cluster:     cluster1,
			ClusterName: cluster1.Name,
			DeployConfig: DeployConfig{
				Env: Env{
					"CLUSTER_LONG_NAME": "Cluster One",
					"PRESENT":           "here",
				},
			},
		},
	)

	skewMs := NewManifests(
		&Manifest{
			Source: project1,
			Deployments: DeploySpecs{
				"cluster-2": { // != cluster-1
					Version: semv.MustParse("2.0.0"),
					DeployConfig: DeployConfig{
						Env: Env{
							"CLUSTER_LONG_NAME": "Cluster One",
							"PRESENT":           "here",
						},
					},
				},
			},
		},
	)
	ms, err := deps.PutbackManifests(defs, skewMs)
	assert.NoError(t, err)
	m, yes := ms.Any(func(*Manifest) bool { return true })
	assert.True(t, yes)
	assert.NotContains(t, m.Deployments["cluster-1"].Env, "CLUSTER_LONG_NAME")
	assert.Contains(t, m.Deployments["cluster-1"].Env, "PRESENT")
}

func TestDeployments_PutbackManifestOriginalLacksEnvElides(t *testing.T) {
	defs := makeTestDefs()
	deps := NewDeployments(
		&Deployment{
			SourceID: project1.SourceID(semv.MustParse("2.0.0")),

			Cluster:     cluster1,
			ClusterName: cluster1.Name,
			DeployConfig: DeployConfig{
				Env: Env{
					"CLUSTER_LONG_NAME": "Cluster One",
					"PRESENT":           "here",
				},
			},
		},
	)

	skewMs := NewManifests(
		&Manifest{
			Source: project1,
			Deployments: DeploySpecs{
				"cluster-1": {
					Version: semv.MustParse("2.0.0"),
					DeployConfig: DeployConfig{
						Env: Env{
							//"CLUSTER_LONG_NAME": "Cluster One",
							"PRESENT": "here",
						},
					},
				},
			},
		},
	)
	ms, err := deps.PutbackManifests(defs, skewMs)
	assert.NoError(t, err)
	m, yes := ms.Any(func(*Manifest) bool { return true })
	assert.True(t, yes)
	assert.NotContains(t, m.Deployments["cluster-1"].Env, "CLUSTER_LONG_NAME")
	assert.Contains(t, m.Deployments["cluster-1"].Env, "PRESENT")
}

func TestDeployments_PutbackManifestHasEnvRetains(t *testing.T) {
	defs := makeTestDefs()
	deps := NewDeployments(
		&Deployment{
			SourceID: project1.SourceID(semv.MustParse("2.0.0")),

			Cluster:     cluster1,
			ClusterName: cluster1.Name,
			DeployConfig: DeployConfig{
				Env: Env{
					"CLUSTER_LONG_NAME": "Cluster One",
					"PRESENT":           "here",
				},
			},
		},
	)

	skewMs := NewManifests(
		&Manifest{
			Source: project1,
			Deployments: DeploySpecs{
				"cluster-1": {
					Version: semv.MustParse("2.0.0"),
					DeployConfig: DeployConfig{
						Env: Env{
							"CLUSTER_LONG_NAME": "Cluster One",
							"PRESENT":           "here",
						},
					},
				},
			},
		},
	)
	ms, err := deps.PutbackManifests(defs, skewMs)
	assert.NoError(t, err)
	m, yes := ms.Any(func(*Manifest) bool { return true })
	assert.True(t, yes)
	assert.Contains(t, m.Deployments["cluster-1"].Env, "CLUSTER_LONG_NAME")
	assert.Contains(t, m.Deployments["cluster-1"].Env, "PRESENT")
}

func compareDeployments(t *testing.T, expectedDeployments, actualDeployments Deployments) {
	exSnap := expectedDeployments.Snapshot()
	if len(actualDeployments.Snapshot()) != len(exSnap) {
		t.Error("deployments different lengths")
	}
	for id, expected := range exSnap {
		actual, ok := actualDeployments.Get(id)
		if !ok {
			t.Errorf("missing deployment %q", id)
			continue
		}
		if !actual.Equal(expected) {
			t.Errorf("\n\ngot:\n%v\n\nwant:\n%v\n", jsonDump(actual), jsonDump(expected))
		}
	}
}

func compareManifests(t *testing.T, expectedManifests, actualManifests Manifests) {
	actualLen := actualManifests.Len()
	expectedLen := expectedManifests.Len()
	if actualLen != expectedLen {
		t.Fatalf("got %d manifests; want %d", actualLen, expectedLen)
	}
	for _, mid := range expectedManifests.Keys() {
		expected, ok := expectedManifests.Get(mid)
		if !ok {
			t.Errorf("missing expected manifest %q", mid)
			continue
		}
		actual, ok := actualManifests.Get(mid)
		if !ok {
			t.Errorf("missing manifest %q", mid)
			continue
		}
		different, differences := actual.Diff(expected)
		if different {
			t.Errorf("manifests not as expected: \n  %s", strings.Join(differences, "\n  "))
			continue
		}
		// Check all expected DeploySpecs are in actual.
		for clusterName := range expected.Deployments {
			did := DeploymentID{Cluster: clusterName, ManifestID: expected.ID()}
			_, ok := actual.Deployments[clusterName]
			if !ok {
				t.Errorf("deployment %q missing", did)
			}
		}
		// Check actual contains only the expected DeploySpecs.
		for clusterName := range actual.Deployments {
			did := DeploymentID{Cluster: clusterName, ManifestID: actual.ID()}
			_, ok := expected.Deployments[clusterName]
			if !ok {
				t.Errorf("extra deployment %q", did)
			}
		}
	}
}

func jsonDump(v interface{}) string { b, _ := json.MarshalIndent(v, "", "  "); return string(b) }
