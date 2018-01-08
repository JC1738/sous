// Package storage is responsible for the persistent storage of state.
//
// Sous state is stored in a file hierarchy like this:
//
//     /
//         defs.yaml
//         manifests/
//             github.com/
//                 username/
//                     reponame/
//                         dirname/
//                             subdirname.yaml
package storage

import (
	"fmt"
	"sort"

	"github.com/opentable/hy"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/yaml"
	"github.com/pkg/errors"
)

type (
	// DiskStateManager implements StateReader and StateWriter using disk
	// storage as its back-end.
	DiskStateManager struct {
		BaseDir string
		Codec   *hy.Codec
	}
)

// NewDiskStateManager returns a new DiskStateManager configured to read and
// write from a filesystem tree containing YAML files.
func NewDiskStateManager(baseDir string) *DiskStateManager {
	c := hy.NewCodec(func(c *hy.Codec) {
		c.FileExtension = "yaml"
		c.MarshalFunc = yaml.Marshal
		c.UnmarshalFunc = yaml.Unmarshal
	})
	return &DiskStateManager{Codec: c, BaseDir: baseDir}
}

func repairState(s *sous.State) error {

	for _, m := range s.Manifests.Snapshot() {
		sort.Strings(m.Owners)
	}

	logging.Log.Vomit.Printf("Validating State")
	flaws := s.Validate()

	logging.Log.Vomit.Printf("Repairing State: %d flaws", len(flaws))
	_, es := sous.RepairAll(flaws)

	if len(es) > 0 {
		strs := []string{}
		for _, e := range es {
			strs = append(strs, e.Error())
		}
		return errors.Errorf("Couldn't repair state: %v", strs)
	}
	return nil
}

// ReadState loads the entire intended state of the world from a dir.
func (dsm *DiskStateManager) ReadState() (*sous.State, error) {
	// TODO: Allow state dir to be passed as flag in sous/cli.
	// TODO: Consider returning a error to indicate if the state dir exists at all.
	logging.Log.Vomit.Printf("Reading state from disk")
	s := sous.NewState()
	err := dsm.Codec.Read(dsm.BaseDir, s)
	if err != nil {
		return s, err
	}

	// XXX Move to validation
	if s.Defs.Clusters == nil {
		return s, nil // errors.Errorf("no clusters defined in %s", dsm.baseDir)
	}
	// XXX Move to validation
	for _, k := range s.Manifests.Keys() {
		m, _ := s.Manifests.Get(k)
		if m == nil {
			return nil, fmt.Errorf("manifest %q is nil", k)
		}
		for clusterName := range m.Deployments {
			if _, ok := s.Defs.Clusters[clusterName]; !ok {
				return s, errors.Errorf("cluster %q not defined (from manifest %q)",
					clusterName, k)
			}
		}
	}
	if e := repairState(s); e != nil {
		return nil, e
	}
	return s, nil
}

// WriteState records the entire intended state of the world to a dir.
func (dsm *DiskStateManager) WriteState(s *sous.State, u sous.User) error {
	if e := repairState(s); e != nil {
		return e
	}
	logging.Log.Vomit.Printf("Writing state to disk")
	return dsm.Codec.Write(dsm.BaseDir, s)
}
