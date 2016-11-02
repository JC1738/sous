package sous

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
)

// Deployments returns all deployments described by the state.
func (s *State) Deployments() (Deployments, error) {
	ds := NewDeployments()
	for _, m := range s.Manifests.Snapshot() {
		deployments, err := s.DeploymentsFromManifest(m)
		if err != nil {
			return ds, err
		}
		conflict, ok := ds.AddAll(deployments)
		if !ok {
			return ds, fmt.Errorf("conflicting deploys: %s", conflict)
		}
	}
	for _, id := range ds.Keys() {
		d, _ := ds.Get(id)
		for name, val := range d.Cluster.Env {
			if _, ok := d.Env[name]; ok {
				continue
			}
			d.Env[name] = string(val)
		}
		cluster, ok := s.Defs.Clusters[d.ClusterName]
		if !ok {
			return ds, errors.Errorf("cluster %q is not described in defs.yaml (but specified in manifest %q)",
				d.ClusterName, id.ManifestID)
		}
		if cluster == nil {
			return ds, errors.Errorf("cluster %q is nil, check defs.yaml", d.ClusterName)
		}
		d.Cluster = cluster
	}
	return ds, nil
}

// Manifests creates manifests from deployments.
func (ds Deployments) Manifests(defs Defs) (Manifests, error) {
	ms := NewManifests()
	for _, k := range ds.Keys() {
		d, _ := ds.Get(k)
		if d.ClusterName == "" {
			return ms, errors.Errorf("invalid deploy ID (no cluster name)")
		}
		if d.Cluster == nil {
			cluster, ok := defs.Clusters[d.ClusterName]
			if !ok {
				return ms, errors.Errorf("cluster %q is not described in defs.yaml", d.ClusterName)
			}
			d.Cluster = cluster
		}
		// Lookup the current manifest for this deployment.
		mid := d.ManifestID()
		m, ok := ms.Get(mid)
		if !ok {
			m = &Manifest{Deployments: DeploySpecs{}}
			for o := range d.Owners {
				// TODO: de-dupe or use a set on manifests.
				m.Owners = append(m.Owners, o)
			}
			m.SetID(mid)
		}
		spec := DeploySpec{
			Version:      d.SourceID.Version,
			DeployConfig: d.DeployConfig.Clone(),
		}
		for k, v := range spec.DeployConfig.Env {
			clusterVal, ok := d.Cluster.Env[k]
			if !ok {
				continue
			}
			if string(clusterVal) == v {
				delete(spec.DeployConfig.Env, k)
			}
		}
		m.Deployments[d.ClusterName] = spec
		m.Kind = d.Kind
		global, pruned := gatherDeploySpecs(m.Deployments)
		var zeroDeploySpec DeploySpec
		if !global.Equal(zeroDeploySpec) {
			m.Deployments = DeploySpecs{}
			m.Deployments["Global"] = global
			for _, sp := range pruned {
				m.Deployments[sp.clusterName] = sp
			}
		}
		ms.Set(mid, m)
	}

	return ms, nil
}

// DeploymentsFromManifest returns all deployments described by a single
// manifest, in terms of the wider state (i.e. global and cluster definitions
// and configuration).
func (s *State) DeploymentsFromManifest(m *Manifest) (Deployments, error) {
	ds := NewDeployments()
	var inherit []DeploySpec
	if global, ok := m.Deployments["Global"]; ok {
		inherit = append(inherit, global)
		delete(m.Deployments, "Global")
	}

	for clusterName, spec := range m.Deployments {
		cluster, ok := s.Defs.Clusters[clusterName]
		if !ok {
			return ds, errors.Errorf("cluster %q not described in defs.yaml", clusterName)
		}
		spec.clusterName = cluster.BaseURL
		d, err := BuildDeployment(s, m, clusterName, spec, inherit)
		if err != nil {
			return ds, err
		}
		ds.Add(d)
	}
	return ds, nil
}

// BuildDeployment constructs a deployment out of a Manifest.
func BuildDeployment(s *State, m *Manifest, nick string, spec DeploySpec, inherit []DeploySpec) (*Deployment, error) {
	ownMap := OwnerSet{}
	for i := range m.Owners {
		ownMap.Add(m.Owners[i])
	}
	ds := flattenDeploySpecs(append([]DeploySpec{spec}, inherit...))
	return &Deployment{
		ClusterName:  nick,
		Cluster:      s.Defs.Clusters[nick],
		DeployConfig: ds.DeployConfig,
		Flavor:       m.Flavor,
		Owners:       ownMap,
		Kind:         m.Kind,
		SourceID:     m.Source.SourceID(ds.Version),
	}, nil
}

func gatherDeploySpecs(dss DeploySpecs) (global DeploySpec, pruned []DeploySpec) {
	var dcs []DeployConfig
	for _, s := range dss {
		dcs = append(dcs, s.DeployConfig)
	}
	gc, pcs := gatherDeployConfigs(dcs)
	gatherVersion := true

	for _, s := range dss[1:] {
		if dss[0].Version != s.Version {
			gatherVersion = false
		}
	}
	global = DeploySpec{DeployConfig: gc}
	if gatherVersion {
		global.Version = dss[0].Version
	}
	pruned = make([]DeploySpec, len(dss))
	for idx := range dss {
		pruned[idx] = DeploySpec{
			DeployConfig: pcs[idx],
			clusterName:  dss[idx].clusterName,
		}
		if !gatherVersion {
			pruned[idx].Version = dss[idx].Version
		}
	}
	return
}

func flattenDeploySpecs(dss []DeploySpec) DeploySpec {
	var dcs []DeployConfig
	for _, s := range dss {
		dcs = append(dcs, s.DeployConfig)
	}
	ds := DeploySpec{DeployConfig: flattenDeployConfigs(dcs)}
	var zeroVersion semv.Version
	for _, s := range dss {
		if s.Version != zeroVersion {
			ds.Version = s.Version
			break
		}
	}
	/*
		DSs have to be unique by clusterName, nicht war?
			for _, s := range dss {
				if s.clusterName != "" {
					ds.clusterName = s.clusterName
					break
				}
			}
	*/
	return ds
}
