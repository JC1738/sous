package sous

import "fmt"

type (
	// A DeployablePair is a pair of deployables, describing a "before and after"
	// situation, where the Prior Deployable is the known state and the Post
	// Deployable is the desired state.
	DeployablePair struct {
		Prior, Post  *Deployable
		name         DeploymentID
		ExecutorData interface{}
	}

	// DeployablePairKind describes the disposition of a DeployablePair
	DeployablePairKind int
)

const (
	// SameKind prior is unchanged from post
	SameKind DeployablePairKind = iota
	// AddedKind means an added deployable - there's no prior
	AddedKind
	// RemovedKind means a removed deployable - no post
	RemovedKind
	// ModifiedKind means modified deployable - post and prior are different
	ModifiedKind
)

// Kind returns the kind of the pair.
func (dp *DeployablePair) Kind() DeployablePairKind {
	switch {
	default:
		return SameKind
	//case dp.Prior == nil && dp.Post == nil:
	//	panic("nil, nil deployable pair")
	case dp.Prior == nil:
		return AddedKind
	case dp.Post == nil:
		return RemovedKind
	case len(dp.Diffs()) != 0:
		return ModifiedKind
	}
}

// Diffs returns the differences from Prior to Post.
func (dp *DeployablePair) Diffs() Differences {
	prior, post := dp.Prior, dp.Post
	_, diffs := prior.Diff(post.Deployment)
	if prior.Status != post.Status {
		diffs = append(diffs, fmt.Sprintf("status prior: %s; post: %s",
			prior.Status, post.Status))
	}
	return diffs
}

// SameResolution returns a DiffResolution indicating that there is no intended
// change. The deployment may either be stable or in the process of being
// deployed.
func (dp *DeployablePair) SameResolution() DiffResolution {
	dep := dp.Prior
	desc := StableDiff
	if dep.Status == DeployStatusPending {
		desc = ComingDiff
	}
	var err *ErrorWrapper
	if dep.Status == DeployStatusFailed {
		err = WrapResolveError(&FailedStatusError{})
	}
	return DiffResolution{
		DeploymentID: dep.ID(),
		Desc:         desc,
		Error:        err,
	}
}
