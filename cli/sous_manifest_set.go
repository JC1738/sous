package cli

import (
	"flag"
	"io/ioutil"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/pkg/errors"
	"github.com/samsalisbury/yaml"
)

type SousManifestSet struct {
	config.DeployFilterFlags
	graph.TargetManifestID
	*sous.State
	graph.StateWriter
	graph.InReader
	*sous.ResolveFilter
	*sous.LogSet
	User sous.User
}

func init() { ManifestSubcommands["set"] = &SousManifestSet{} }

const sousManifestSetHelp = `replace a deployment manifest

do note: this does *replace* the manifest;
there's some validation, but you can make drastic changes easily
`

func (*SousManifestSet) Help() string { return sousManifestHelp }

func (smg *SousManifestSet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &smg.DeployFilterFlags, ManifestFilterFlagsHelp)
}

func (smg *SousManifestSet) RegisterOn(psy Addable) {
	psy.Add(&smg.DeployFilterFlags)
}

func (smg *SousManifestSet) Execute(args []string) cmdr.Result {
	mid := sous.ManifestID(smg.TargetManifestID)

	_, present := smg.State.Manifests.Get(mid)
	if !present {
		return EnsureErrorResult(errors.Errorf("No manifest matched by %v yet. See `sous init`", smg.ResolveFilter))
	}

	yml := sous.Manifest{}
	bytes, err := ioutil.ReadAll(smg.InReader)
	if err != nil {
		return EnsureErrorResult(err)
	}
	err = yaml.Unmarshal(bytes, &yml)
	if err != nil {
		return EnsureErrorResult(err)
	}
	smg.Vomit.Print(spew.Sdump(yml))
	smg.State.Manifests.Set(mid, &yml)
	err = smg.StateWriter.WriteState(smg.State, smg.User)
	if err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success()
}
