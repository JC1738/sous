package docker

import (
	"bytes"
	"fmt"
	"io"

	"github.com/nyarly/inlinefiles/templatestore"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
)

//go:generate inlinefiles --vfs=templateVFS tmpl/ templates_vfs.go

type (
	// Builder represents a single build of a project.
	Builder struct {
		ImageMapper               sous.Inserter
		DockerRegistryHost        string
		SourceShell, ScratchShell shell.Shell
		Pack                      sous.Buildpack
		registry                  docker_registry.Client
	}
	// BuildTarget represents a single target within a Build.
	BuildTarget interface {
		BuildImage()
		BuildContainer()
	}
)

// NewBuilder creates a new build using source code in the working
// directory of sourceShell, and using the working dir of scratchShell as
// temporary storage.
func NewBuilder(nc sous.Inserter, drh string, reg docker_registry.Client, sourceShell, scratchShell shell.Shell) (*Builder, error) {
	b := &Builder{
		DockerRegistryHost: drh,
		SourceShell:        sourceShell,
		ScratchShell:       scratchShell,
		registry:           reg,
	}

	files, err := scratchShell.List()
	if err != nil {
		return nil, err
	}

	if len(files) != 0 {
		return nil, fmt.Errorf("scratch dir %s was not empty", scratchShell.Dir())
	}

	return b, nil
}

func (b *Builder) debug(msg string) {
	Log.Debug.Printf(msg)
}

func (b *Builder) info(msg string) {
	Log.Info.Printf(msg)
}

// CheckExists implements Registrar on Builder
func (b *Builder) CheckExists(bc *sous.BuildContext) bool {
	imageName = b.VersionTag(bc.Version(), "")

	_, err := b.registry.GetImageMetadata(imageName, "")
	if err != nil {
		return false
	}
	return true

}

// Register registers the build artifact to the the registry
func (b *Builder) Register(br *sous.BuildResult, bc *sous.BuildContext) error {
	err := b.pushToRegistry(br, bc)
	if err != nil {
		return err
	}

	return b.recordName(br, bc)
}

// ApplyMetadata implements sous.Labeller on Builder.
// It applies container metadata etc. to a container.
func (b *Builder) ApplyMetadata(br *sous.BuildResult, bc *sous.BuildContext) error {
	return b.applyMetadata(br, "", bc)
}

func (b *Builder) applyMetadata(br *sous.BuildResult, kind string, bc *sous.BuildContext) error {
	br.VersionName = b.VersionTag(bc.Version(), kind)
	br.RevisionName = b.RevisionTag(bc.Version(), kind)

	c := b.SourceShell.Cmd("docker", "build", "-t", br.VersionName, "-t", br.RevisionName, "-")
	bf := b.metadataDockerfile(br, bc)
	c.SetStdin(bf)

	for kind, rez := range br.ExtraResults {
		err := b.applyMetadata(rez, kind, bc)
		if err != nil {
			return err
		}
	}

	return c.Succeed()
}

func (b *Builder) metadataDockerfile(br *sous.BuildResult, bc *sous.BuildContext) io.Reader {
	bf := bytes.Buffer{}
	sv := bc.Version()
	md, err := templatestore.LoadText(templateVFS, "metadata", "metadataDockerfile.tmpl")
	if err != nil {
		panic(err)
	}

	md.Execute(&bf, struct {
		ImageID    string
		Labels     map[string]string
		Advisories []string
	}{
		br.ImageID,
		Labels(sv),
		br.Advisories,
	})
	return &bf
}

// pushToRegistry sends the built image to the registry
func (b *Builder) pushToRegistry(br *sous.BuildResult, bc *sous.BuildContext) error {
	for _, rez := range br.ExtraResults {
		if err := b.pushToRegistry(rez, bc); err != nil {
			return err
		}
	}

	verr := b.SourceShell.Run("docker", "push", br.VersionName)
	rerr := b.SourceShell.Run("docker", "push", br.RevisionName)

	if verr == nil {
		return rerr
	}
	return verr
}

// recordName inserts metadata about the newly built image into our local name cache
func (b *Builder) recordName(br *sous.BuildResult, bc *sous.BuildContext) error {
	sv := bc.Version()
	in := br.VersionName
	b.SourceShell.ConsoleEcho(fmt.Sprintf("[recording \"%s\" as the docker name for \"%s\"]", in, sv.String()))
	var qs []sous.Quality
	for _, adv := range br.Advisories {
		qs = append(qs, sous.Quality{Name: adv, Kind: "advisory"})
	}
	return b.ImageMapper.Insert(sv, in, "", qs)
}

// VersionTag computes an image tag from a SourceVersion's version
func (b *Builder) VersionTag(v sous.SourceID, kind string) string {
	return versionTag(b.DockerRegistryHost, v, kind)
}

// RevisionTag computes an image tag from a SourceVersion's revision id
func (b *Builder) RevisionTag(v sous.SourceID, kind string) string {
	return revisionTag(b.DockerRegistryHost, v, kind)
}
