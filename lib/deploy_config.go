package sous

import (
	"fmt"

	"github.com/pkg/errors"
)

type (
	// DeployConfig represents the configuration of a deployment's tasks,
	// in a specific cluster. i.e. their resources, environment, and the number
	// of instances.
	DeployConfig struct {
		// Resources represents the resources each instance of this software
		// will be given by the execution environment.
		Resources Resources `yaml:",omitempty" validate:"keys=nonempty,values=nonempty"`
		// Metadata stores values about deployments for outside applications to use
		Metadata Metadata `yaml:",omitempty" validate:"keys=nonempty,values=nonempty"`
		// Env is a list of environment variables to set for each instance of
		// of this deployment. It will be checked for conflict with the
		// definitions found in State.Defs.EnvVars, and if not in conflict
		// assumes the greatest priority.
		Args []string `yaml:",omitempty" validate:"values=nonempty"`
		Env  `yaml:",omitempty" validate:"keys=nonempty,values=nonempty"`
		// NumInstances is a guide to the number of instances that should be
		// deployed in this cluster, note that the actual number may differ due
		// to decisions made by Sous. If set to zero, Sous will decide how many
		// instances to launch.
		NumInstances int

		// Volumes lists the volume mappings for this deploy
		Volumes Volumes
	}

	// Env is a mapping of environment variable name to value, used to provision
	// single instances of an application.
	Env map[string]string

	// Metadata represents an opaque map of metadata - Sous is agnostic about
	// its contents, except to validate it against the top level schema
	Metadata map[string]string

	// NilVolumeFlaw is used when DeployConfig.Volumes contains a nil.
	NilVolumeFlaw struct {
		*DeployConfig
	}
)

// Validate implements Flawed for State
func (dc *DeployConfig) Validate() []Flaw {
	var flaws []Flaw

	for _, v := range dc.Volumes {
		if v == nil {
			flaws = append(flaws, &NilVolumeFlaw{DeployConfig: dc})
			break
		}
	}
	rezs := dc.Resources
	if dc.Resources == nil {
		flaws = append(flaws, NewFlaw("No Resources set for DeployConfig",
			func() error { dc.Resources = make(Resources); return nil }))
		rezs = make(Resources)
	}

	flaws = append(flaws, rezs.Validate()...)

	for _, f := range flaws {
		f.AddContext("deploy config", dc)
	}

	return flaws
}

// AddContext simply discards all context - NilVolumeFlaw doesn't need it
func (nvf *NilVolumeFlaw) AddContext(string, interface{}) {
}

// Repair removes any nil entries in DeployConfig.Volumes.
func (nvf *NilVolumeFlaw) Repair() error {
	newVs := nvf.DeployConfig.Volumes[:0]
	for _, v := range nvf.DeployConfig.Volumes {
		if v != nil {
			newVs = append(newVs, v)
		}
	}
	nvf.DeployConfig.Volumes = newVs
	return nil
}

// Repair implements Flawed for State
func (dc *DeployConfig) Repair(fs []Flaw) error {
	return errors.Errorf("Can't do nuffin with flaws yet")
}

func (dc *DeployConfig) String() string {
	return fmt.Sprintf("#%d %+v : %+v %+v", dc.NumInstances, dc.Resources, dc.Env, dc.Volumes)
}

// Equal is used to compare DeployConfigs
func (dc *DeployConfig) Equal(o DeployConfig) bool {
	Log.Vomit.Printf("%+ v ?= %+ v", dc, o)
	diff, _ := dc.Diff(o)
	return !diff
}

// Diff returns a list of differences between this and the other DeployConfig.
func (dc *DeployConfig) Diff(o DeployConfig) (bool, []string) {
	var diffs []string
	if dc.NumInstances != o.NumInstances {
		diffs = append(diffs, fmt.Sprintf("number of instances; this: %d; other: %d", dc.NumInstances, o.NumInstances))
	}
	// Only compare contents if length of either > 0.
	// This makes nil equal to zero-length map.
	if len(dc.Env) != 0 || len(o.Env) != 0 {
		if !dc.Env.Equal(o.Env) {
			diffs = append(diffs, fmt.Sprintf("env; this: %v; other: %v", dc.Env, o.Env))
		}
	}
	// Only compare contents if length of either > 0.
	if len(dc.Resources) != 0 || len(o.Resources) != 0 {
		if !dc.Resources.Equal(o.Resources) {
			diffs = append(diffs, fmt.Sprintf("resources; this: %v; other: %v", dc.Resources, o.Resources))
		}
	}
	// Only compare contents if length of either > 0.
	if len(dc.Volumes) != 0 || len(o.Volumes) != 0 {
		if !dc.Volumes.Equal(o.Volumes) {
			diffs = append(diffs, fmt.Sprintf("volumes; this: %v; other: %v", dc.Volumes, o.Volumes))
		}
	}
	// TODO: Compare Args
	return len(diffs) == 0, diffs
}

// Clone returns a deep copy of this DeployConfig.
func (dc DeployConfig) Clone() (c DeployConfig) {
	c.NumInstances = dc.NumInstances
	c.Args = make([]string, len(dc.Args))
	copy(dc.Args, c.Args)
	c.Env = make(Env)
	for k, v := range dc.Env {
		c.Env[k] = v
	}
	c.Resources = make(Resources)
	for k, v := range dc.Resources {
		c.Resources[k] = v
	}
	c.Volumes = make(Volumes, len(dc.Volumes))
	copy(dc.Volumes, c.Volumes)
	return
}

// Equal compares Envs
func (e Env) Equal(o Env) bool {
	Log.Vomit.Printf("Envs: %+ v ?= %+ v", e, o)
	if len(e) != len(o) {
		Log.Vomit.Printf("Envs: %+ v != %+ v (%d != %d)", e, o, len(e), len(o))
		return false
	}

	for name, value := range e {
		if ov, ok := o[name]; !ok || ov != value {
			Log.Vomit.Printf("Envs: %+ v != %+ v [%q] %q != %q", e, o, name, value, ov)
			return false
		}
	}
	Log.Vomit.Printf("Envs: %+ v == %+ v !", e, o)
	return true
}

func gatherDeployConfigs(dcs []DeployConfig) (global DeployConfig, pruned []DeployConfig) {
	global = dcs[0].Clone()
	var niVary, volsVary, argsVary, rezVary, envVary bool

	for _, c := range dcs[1:] {
		if c.NumInstances != global.NumInstances {
			niVary = true
		}
		if !c.Volumes.Equal(global.Volumes) {
			volsVary = true
		}
		if !stringSlicesEqual(c.Args, global.Args) {
			argsVary = true
		}
		if len(global.Resources) != len(c.Resources) {
			rezVary = true
		} else {
			for n, v := range c.Resources {
				if gv, set := global.Resources[n]; !set || v != gv {
					rezVary = true
				}
			}
		}
		if len(global.Env) != len(c.Env) {
			envVary = true
		} else {
			for n, v := range c.Env {
				if gv, set := global.Env[n]; !set || v != gv {
					envVary = true
				}
			}
		}
	}

	if niVary {
		global.NumInstances = 0
	}
	if volsVary {
		global.Volumes = Volumes{}
	}
	if argsVary {
		global.Args = []string{}
	}
	if rezVary {
		global.Resources = Resources{}
	}
	if envVary {
		global.Env = Env{}
	}

	pruned = make([]DeployConfig, len(dcs))
	for idx := range dcs {
		pruned[idx] = dcs[idx].Clone()
		if !niVary {
			pruned[idx].NumInstances = 0
		}
		if !volsVary {
			pruned[idx].Volumes = Volumes{}
		}
		if !argsVary {
			pruned[idx].Args = []string{}
		}
		if !rezVary {
			pruned[idx].Resources = Resources{}
		}
		if !envVary {
			pruned[idx].Env = Env{}
		}
	}
	return
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func flattenDeployConfigs(dcs []DeployConfig) DeployConfig {
	dc := DeployConfig{
		Resources: make(Resources),
		Env:       make(Env),
	}
	for _, c := range dcs {
		if c.NumInstances != 0 {
			dc.NumInstances = c.NumInstances
			break
		}
	}
	for _, c := range dcs {
		if len(c.Volumes) != 0 {
			dc.Volumes = c.Volumes
			break
		}
	}
	for _, c := range dcs {
		if len(c.Args) != 0 {
			dc.Args = c.Args
			break
		}
	}
	for _, c := range dcs {
		for n, v := range c.Resources {
			if _, set := dc.Resources[n]; !set {
				dc.Resources[n] = v
			}
		}
		for n, v := range c.Env {
			if _, set := dc.Env[n]; !set {
				dc.Env[n] = v
			}
		}
	}
	return dc
}
