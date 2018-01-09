package sous

import (
	"sync"
)

type (
	// A DeployableChans is a bundle of channels describing actions to take on a
	// cluster
	DeployableChans struct {
		Pairs chan *DeployablePair
		Errs  chan *DiffResolution
		sync.WaitGroup
	}

	// DeployablePairs is a list of DeployablePair
	DeployablePairs []*DeployablePair
)

// NewDeployableChans returns a new DeployableChans with channels buffered to
// size.
func NewDeployableChans(size ...int) *DeployableChans {
	var s int
	if len(size) > 0 {
		s = size[0]
	}
	return &DeployableChans{
		Pairs: make(chan *DeployablePair, s),
		Errs:  make(chan *DiffResolution, s),
	}
}

// Close closes all the channels in a DeployableChans in a single action
func (d *DeployableChans) Close() {
	close(d.Pairs)
	close(d.Errs)
}

func (d *DeployableChans) collect() DeployablePairs {
	ds := DeployablePairs{}

	for m := range d.Pairs {
		ds = append(ds, m)
	}
	return ds
}

// Collect returns a collected list of DeployablePairs represented by this DeployableChans
func (d *DeployableChans) Collect() DeployablePairs {
	return d.collect()
}

// ID returns the ID of this DeployablePair.
func (dp *DeployablePair) ID() DeploymentID {
	return dp.name
}

func (kind DeployablePairKind) String() string {
	switch kind {
	default:
		return "unknown"
	case SameKind:
		return "same"
	case AddedKind:
		return "added"
	case RemovedKind:
		return "removed"
	case ModifiedKind:
		return "modified"
	}
}

// ResolveVerb provides an imperative verb describing the resolution action
// required to resolve this kind of deployable pair.
func (kind DeployablePairKind) ResolveVerb() string {
	switch kind {
	default:
		return ""
	case SameKind:
		return "take no action"
	case AddedKind:
		return "create new deployment"
	case RemovedKind:
		return "delete existing deployment"
	case ModifiedKind:
		return "update existing deployment"
	}
}
