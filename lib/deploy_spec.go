package sous

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
)

type (
	// DeploySpecs is a collection of Deployments associated with a manifest.
	DeploySpecs map[string]DeploySpec

	// DeploySpec is the interface to describe a cluster-wide deployment
	// of an application described by a Manifest. Together with the manifest,
	// one can assemble full Deployments.
	//
	// Unexported fields in DeploymentSpec are not intended to be serialised
	// to/from yaml, but are useful when set internally.
	DeploySpec struct {
		// DeployConfig contains config information for this deployment, see
		// DeployConfig.
		DeployConfig `yaml:",inline"`
		// Version is a semantic version with the following properties:
		//
		//     1. The major/minor/patch/pre-release fields exist as a tag in
		//        the source code repository containing this application.
		//     2. The metadata field is the full revision ID of the commit
		//        which the tag in 1. points to.
		Version semv.Version `validate:"nonzero"`
		// clusterName is the name of the cluster this deployment belongs to. Upon
		// parsing the Manifest, this will be set to the key in
		// Manifests.Deployments which points at this Deployment.
		clusterName string
	}
)

func (spec DeploySpec) String() string {
	return fmt.Sprintf("%v %s", spec.Version, spec.DeployConfig.String())
}

// Validate implements Flawed for State
func (spec DeploySpec) Validate() []Flaw {
	return spec.DeployConfig.Validate()
}

// Repair implements Flawed for State
func (spec DeploySpec) Repair(fs []Flaw) error {
	return errors.Errorf("Can't do nuffin with flaws yet")
}

// Clone returns a deep copy of this DeploySpec.
func (spec DeploySpec) Clone() DeploySpec {
	spec.DeployConfig = spec.DeployConfig.Clone()
	return spec
}

// Equal returns true if other equals spec.
func (spec DeploySpec) Equal(other DeploySpec) bool {
	different, _ := spec.Diff(other)
	return !different
}

// Diff returns true and a list of differences if spec is different to other.
// Otherwise returns false, nil.
func (spec DeploySpec) Diff(other DeploySpec) (bool, []string) {
	var diffs []string
	diff := func(format string, a ...interface{}) { diffs = append(diffs, fmt.Sprintf(format, a...)) }
	if !spec.Version.Equals(other.Version) {
		diff("version; this: %q; other: %q", spec.Version, other.Version)
	}
	_, configDiffs := spec.DeployConfig.Diff(other.DeployConfig)
	for _, d := range configDiffs {
		diff(d)
	}
	return len(diffs) != 0, diffs
}

func (spec DeploySpec) isZero() bool {
	var zeroSpec DeploySpec
	return spec.Equal(zeroSpec)
}
