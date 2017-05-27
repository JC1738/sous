package sous

import (
	"fmt"
	"strings"
	"time"
)

type (
	// A Selector selects the buildpack for a given build context
	Selector interface {
		SelectBuildpack(*BuildContext) (Buildpack, error)
	}

	// Labeller defines a container-based build system.
	Labeller interface {
		ApplyMetadata(*BuildResult, *BuildContext) error
	}

	// Registrar defines the interface to register build results to be deployed
	// later
	Registrar interface {
		// Register takes a BuildResult and makes it available for the deployment
		// target system to find during deployment
		Register(*BuildResult, *BuildContext) error
		CheckExists(*BuildContext) bool
	}

	// Strpairs is a slice of Strpair.
	Strpairs []Strpair

	// Strpair is a pair of strings.
	Strpair [2]string

	// BuildArtifact describes the actual built binary Sous will deploy
	BuildArtifact struct {
		Name, Type string
		Qualities  []Quality
	}

	// A Quality represents a characteristic of a BuildArtifact that needs to be recorded.
	Quality struct {
		Name string
		// Kind is the the kind of this quality
		// Known kinds include: advisory
		Kind string
	}

	// Buildpack is a set of instructions used to build a particular
	// kind of project.
	Buildpack interface {
		Detect(*BuildContext) (*DetectResult, error)
		Build(*BuildContext, *DetectResult) (*BuildResult, error)
	}

	// DetectResult represents the result of a detection.
	DetectResult struct {
		// Compatible is true when the buildpack is compatible with the source
		// context.
		Compatible bool
		// Description is a human-readable description of what will be built.
		// It may for instance report back the base image that will be used,
		// or detected runtime version etc.
		Description string
		// Data is an arbitrary value. It can be used to pass interesting
		// detected information to the build step.
		Data interface{}
	}
	// BuildResult represents the result of a build made with a Buildpack.
	BuildResult struct {
		ImageID                   string
		VersionName, RevisionName string
		Elapsed                   time.Duration
		ExtraResults              map[string]*BuildResult
		Advisories
	}

	// EchoSelector wraps a buildpack Factory. But why?
	EchoSelector struct {
		Factory func(*BuildContext) (Buildpack, error)
	}
)

// SelectBuildpack tries to select a buildpack for this BuildContext.
func (s *EchoSelector) SelectBuildpack(c *BuildContext) (Buildpack, error) {
	return s.Factory(c)
}

func (br *BuildResult) String() string {
	str := fmt.Sprintf("Built: %q", br.VersionName)
	if len(br.Advisories) > 0 {
		str = str + "\nAdvisories:\n  " + strings.Join(br.Advisories.strings(), "  \n")
	}
	return fmt.Sprintf("%s\nElapsed: %s", str, br.Elapsed)
}

// NewBuildArtifact creates a new BuildArtifact representing a Docker
// image.
func NewBuildArtifact(imageName string, qstrs Strpairs) *BuildArtifact {
	var qs []Quality
	for _, qstr := range qstrs {
		qs = append(qs, Quality{Name: qstr[0], Kind: qstr[1]})
	}

	return &BuildArtifact{Name: imageName, Type: "docker", Qualities: qs}
}
