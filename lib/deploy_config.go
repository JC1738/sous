package sous

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
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
		// Startup containts healthcheck options for this deploy.
		Startup Startup `yaml:",omitempty"`
	}

	// A DeployConfigs is a map from cluster name to DeployConfig
	DeployConfigs map[string]DeployConfig

	// Env is a mapping of environment variable name to value, used to provision
	// single instances of an application.
	Env map[string]string

	// Startup is a struct of options related to container startup.  Members are
	// pointers so that they can be ignored if nil.
	Startup struct {
		CheckReadyURIPath    *string `yaml:",omitempty"`
		CheckReadyURITimeout *int    `yaml:",omitempty"`
		Timeout              *int    `yaml:",omitempty"`
	}

	// Metadata represents an opaque map of metadata - Sous is agnostic about
	// its contents, except to validate it against the top level schema.
	Metadata map[string]string

	// NilVolumeFlaw is used when DeployConfig.Volumes contains a nil.
	NilVolumeFlaw struct {
		*DeployConfig
	}
)

// Validate returns a slice of Flaws.
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

// AddContext simply discards all context - NilVolumeFlaw doesn't need it.
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
	return fmt.Sprintf("#%d %s %+v : %+v %+v", dc.NumInstances, spew.Sprintf("%v", dc.Startup), dc.Resources, dc.Env, dc.Volumes)
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
	// This makes nil equal to zero-length map.
	if len(dc.Metadata) != 0 || len(o.Metadata) != 0 {
		if !dc.Metadata.Equal(o.Metadata) {
			diffs = append(diffs, fmt.Sprintf("metadata; this: %v; other: %v", dc.Metadata, o.Metadata))
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
	diffs = append(diffs, dc.Startup.diff(o.Startup)...)
	// TODO: Compare Args
	return len(diffs) == 0, diffs
}

func (s Startup) diff(o Startup) []string {
	diffs := []string{}
	diff := func(format string, a ...interface{}) { diffs = append(diffs, fmt.Sprintf(format, a...)) }

	if s.CheckReadyURIPath != nil {
		if o.CheckReadyURIPath == nil {
			diff("CheckReadyURIPath; this %q, other empty", *s.CheckReadyURIPath)
		} else if *s.CheckReadyURIPath != *o.CheckReadyURIPath {
			diff("CheckReadyURIPath; this %q, other %q", *s.CheckReadyURIPath, *o.CheckReadyURIPath)
		}
	} else {
		if o.CheckReadyURIPath != nil {
			diff("CheckReadyURIPath; this empty, other %q", *o.CheckReadyURIPath)
		}
	}

	if s.CheckReadyURITimeout != nil {
		if o.CheckReadyURITimeout == nil {
			diff("CheckReadyURITimeout; this %d, other empty", *s.CheckReadyURITimeout)
		} else if *s.CheckReadyURITimeout != *o.CheckReadyURITimeout {
			diff("CheckReadyURITimeout; this %d, other %d", *s.CheckReadyURITimeout, *o.CheckReadyURITimeout)
		}
	} else {
		if o.CheckReadyURITimeout != nil {
			diff("CheckReadyURITimeout; this empty, other %d", *o.CheckReadyURITimeout)
		}
	}

	if s.Timeout != nil {
		if o.Timeout == nil {
			diff("Timeout; this %d, other empty", *s.Timeout)
		} else if *s.Timeout != *o.Timeout {
			diff("Timeout; this %d, other %d", *s.Timeout, *o.Timeout)
		}
	} else {
		if o.Timeout != nil {
			diff("Timeout; this empty, other %d", *o.Timeout)
		}
	}

	return diffs
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
	if dc.Metadata != nil {
		c.Metadata = make(Metadata)
		for k, v := range dc.Metadata {
			c.Metadata[k] = v
		}
	}
	c.Volumes = dc.Volumes.Clone()

	if dc.Startup.CheckReadyURIPath != nil {
		uripath := *dc.Startup.CheckReadyURIPath
		c.Startup.CheckReadyURIPath = &uripath
	}

	if dc.Startup.CheckReadyURITimeout != nil {
		crtimeout := *dc.Startup.CheckReadyURITimeout
		c.Startup.CheckReadyURITimeout = &crtimeout
	}

	if dc.Startup.Timeout != nil {
		timeout := *dc.Startup.Timeout
		c.Startup.Timeout = &timeout
	}

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

// Equal compares Metadatas
func (e Metadata) Equal(o Metadata) bool {
	Log.Vomit.Printf("Metadatas: %+ v ?= %+ v", e, o)
	if len(e) != len(o) {
		Log.Vomit.Printf("Metadatas: %+ v != %+ v (%d != %d)", e, o, len(e), len(o))
		return false
	}

	for name, value := range e {
		if ov, ok := o[name]; !ok || ov != value {
			Log.Vomit.Printf("Metadatas: %+ v != %+ v [%q] %q != %q", e, o, name, value, ov)
			return false
		}
	}
	Log.Vomit.Printf("Metadatas: %+ v == %+ v !", e, o)
	return true
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
		Metadata:  make(Metadata),
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
		for n, v := range c.Metadata {
			if _, set := dc.Metadata[n]; !set {
				dc.Metadata[n] = v
			}
		}

		if dc.Startup.CheckReadyURIPath == nil {
			dc.Startup.CheckReadyURIPath = c.Startup.CheckReadyURIPath
		}
		if dc.Startup.CheckReadyURITimeout == nil {
			dc.Startup.CheckReadyURITimeout = c.Startup.CheckReadyURITimeout
		}
		if dc.Startup.Timeout == nil {
			dc.Startup.Timeout = c.Startup.Timeout
		}
	}
	return dc
}
