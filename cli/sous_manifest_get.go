package cli

import (
	"flag"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/pkg/errors"
	"github.com/samsalisbury/yaml"
)

type SousManifestGet struct {
	config.DeployFilterFlags
	graph.TargetManifestID
	*sous.State
	*sous.LogSet
	graph.OutWriter
	*sous.ResolveFilter
}

func init() { ManifestSubcommands["get"] = &SousManifestGet{} }

const sousManifestGetHelp = `query deployment manifest`

func (*SousManifestGet) Help() string { return sousManifestHelp }

func (smg *SousManifestGet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &smg.DeployFilterFlags, ManifestFilterFlagsHelp)
}

func (smg *SousManifestGet) RegisterOn(psy Addable) {
	psy.Add(&smg.DeployFilterFlags)
}

func (smg *SousManifestGet) Execute(args []string) cmdr.Result {
	mid := sous.ManifestID(smg.TargetManifestID)

	mani, present := smg.State.Manifests.Get(mid)
	if !present {
		return EnsureErrorResult(errors.Errorf("No manifest matched by %v yet. See `sous init`", smg.ResolveFilter))
	}
	smg.Vomit.Print(spew.Sdump(mani))

	yml, err := yaml.Marshal(mani)
	if err != nil {
		return EnsureErrorResult(err)
	}
	smg.OutWriter.Write(yml)
	return cmdr.Success()
}
