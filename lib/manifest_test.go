package sous

import (
	"fmt"
	"testing"

	"github.com/samsalisbury/semv"
)

func TestManifest_Clone(t *testing.T) {
	orignalVersion := semv.MustParse("1")
	crutime := 1234
	original := &Manifest{
		Kind:   ManifestKindService,
		Owners: []string{"some", "owners"},
		Deployments: DeploySpecs{
			"some-cluster": DeploySpec{
				DeployConfig: DeployConfig{
					Resources: Resources{
						"cpus":   "1",
						"memory": "256",
						"ports":  "1",
					},
					NumInstances: 3,
					Startup: Startup{
						CheckReadyURITimeout: &crutime,
					},
				},
				Version: orignalVersion,
			},
		},
	}

	clone := original.Clone()

	if areDiff, diffs := original.Diff(clone); areDiff {
		t.Errorf("Original reports differences from clone: %v", diffs)
	}

	if clone.Kind != ManifestKindService {
		t.Errorf("Kind didn't match orignal")
	}

	if len(clone.Owners) != 2 &&
		clone.Owners[0] != "some" &&
		clone.Owners[1] != "owners" {
		t.Errorf("Owners didn't match orignal")
	}

	if clone.Deployments["some-cluster"].Version != orignalVersion {
		t.Errorf("Deployments didn't match orignal")
	}

	original.Owners = []string{"no", "body"}
	original.Deployments = DeploySpecs{}
	original.Kind = ManifestKindScheduled

	if clone.Kind != ManifestKindService {
		t.Errorf("Kind was changed by mutating original")
	}

	if len(clone.Owners) != 2 &&
		clone.Owners[0] != "some" &&
		clone.Owners[1] != "owners" {
		t.Errorf("Owners was changed by mutating original")
	}

	if clone.Deployments["some-cluster"].Version != orignalVersion {
		t.Errorf("Owners was changed by mutating original")
	}

}

var manifestTests = []struct {
	OriginalManifest, FixedManifest *Manifest
	FlawDesc, RepairError           string
}{
	{
		OriginalManifest: &Manifest{},
		FixedManifest:    &Manifest{Kind: ManifestKindService},
		FlawDesc:         `manifest "" missing Kind`},
	{
		OriginalManifest: &Manifest{Kind: "some invalid kind"},
		FixedManifest:    &Manifest{Kind: "some invalid kind"},
		FlawDesc:         `ManifestKind "some invalid kind" not valid`,
		RepairError:      "unable to repair invalid ManifestKind",
	},
	{
		OriginalManifest: &Manifest{
			Kind: ManifestKindService,
			Deployments: DeploySpecs{
				"some-cluster": DeploySpec{
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus": "1",
							// NOTE: Missing memory.
							"ports": "1",
						},
						NumInstances: 3,
					},
					Version: semv.MustParse("1"),
				},
			},
		},
		FixedManifest: &Manifest{
			Kind: ManifestKindService,
			Deployments: DeploySpecs{
				"some-cluster": DeploySpec{
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus": "1",
							// NOTE: Memory repaired by setting to default.
							"memory": "100",
							"ports":  "1",
						},
						NumInstances: 3,
					},
					Version: semv.MustParse("1"),
				},
			},
		},
		FlawDesc: "Missing resource field \"memory\" for cluster some-cluster",
	},
	{
		// NOTE: This one is valid, hence no FlawDesc.
		OriginalManifest: &Manifest{
			Kind: ManifestKindService,
			Deployments: DeploySpecs{
				"some-cluster": DeploySpec{
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus":   "1",
							"memory": "256",
							"ports":  "1",
						},
						NumInstances: 3,
					},
					Version: semv.MustParse("1"),
				},
			},
		},
	},
}

func TestManifest_Validate(t *testing.T) {
	for i, test := range manifestTests {
		m := test.OriginalManifest
		flaws := m.Validate()
		expectedNumFlaws := 1
		if test.FlawDesc == "" {
			expectedNumFlaws = 0
		}
		if len(flaws) != expectedNumFlaws {
			for _, f := range flaws {
				t.Error(f)
			}
			t.Fatalf("%d: got %d flaws; want %d", i, len(flaws), expectedNumFlaws)
		}
		if test.FlawDesc != "" {
			expectedFlawDesc := test.FlawDesc
			actualFlawDesc := fmt.Sprint(flaws[0])
			if actualFlawDesc != expectedFlawDesc {
				t.Errorf("got flaw desc %q; want %q", actualFlawDesc, expectedFlawDesc)
			}
		}
		if expectedNumFlaws == 0 {
			return
		}
		err := flaws[0].Repair()
		expected := test.RepairError
		if test.RepairError == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil {
			t.Fatalf("got nil; want error %q", expected)
		} else {
			actual := err.Error()
			if actual != expected {
				t.Errorf("got error %q; want %q", actual, expected)
			}
		}
		if test.FixedManifest != nil {
			different, differences := m.Diff(test.FixedManifest)
			if different {
				t.Errorf("repaired manifest not as expected: % #v", differences)
			}
		}
	}
}

func TestManifest_Diff(t *testing.T) {
	a := &Manifest{
		Deployments: DeploySpecs{
			"cluster-a": DeploySpec{},
		},
	}
	b := &Manifest{
		Deployments: DeploySpecs{
			"cluster-b": DeploySpec{},
		},
	}
	different, diffs := a.Diff(b)
	if !different {
		t.Errorf("%# v.Diff(%# v) == %v, %v", a, b, different, diffs)
	}
	if len(diffs) != 2 {
		t.Fatalf("got %d diffs; want %d", len(diffs), 2)
	}
	actual := diffs[0]
	expected := `missing deployment "cluster-a"`
	if actual != expected {
		t.Errorf("diffs[0] == %q; want %q", actual, expected)
	}
	actual = diffs[1]
	expected = `extra deployment "cluster-b"`
	if actual != expected {
		t.Errorf("diffs[1] == %q; want %q", actual, expected)
	}
}
