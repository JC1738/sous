package actions

import (
	"fmt"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/pkg/errors"
)

// Update is the command description for `sous update`
type Update struct {
	Manifest      *sous.Manifest
	GDM           sous.Deployments
	Client        restful.HTTPClient
	ResolveFilter *sous.ResolveFilter
	User          sous.User
	Log           logging.LogSink
}

// Do performs the appropriate update, returning nil on success.
func (u *Update) Do() error {
	mid := u.Manifest.ID()

	sid, err := u.ResolveFilter.SourceID(mid)
	if err != nil {
		return err
	}
	did, err := u.ResolveFilter.DeploymentID(mid)
	if err != nil {
		return err
	}

	gdm, err := updateRetryLoop(u.Log, u.Client, sid, did, u.User)
	if err != nil {
		return err
	}

	// we update the in-memory GDM so that we can poll based on it.
	for k, d := range gdm.Snapshot() {
		u.GDM.Set(k, d)
	}
	return nil
}

// If multiple updates are attempted at once for different clusters, there's
// the possibility that they will collide in their updates, either interleaving
// their GDM retrieve/manifest update operations, or the git pull/push
// server-side. In this case, the disappointed `sous update` should retry, up
// to the number of times of manifests there are defined for this
// SourceLocation
func updateRetryLoop(ls logging.LogSink,
	cl restful.HTTPClient,
	sid sous.SourceID,
	did sous.DeploymentID,
	user sous.User) (sous.Deployments, error) {
	sm := sous.NewHTTPStateManager(cl)

	tryLimit := 2

	mid := did.ManifestID

	start := time.Now()

	for tries := 0; tries < tryLimit; tries++ {
		logging.Deliver(newUpdateBeginMessage(tries, sid, did, user, start), ls)

		var state *sous.State
		var err error
		var ok bool
		var manifest *sous.Manifest
		var gdm sous.Deployments

		if state, err = sm.ReadState(); err != nil {
			return sous.NewDeployments(), err
		}

		if manifest, ok = state.Manifests.Get(mid); !ok {
			err = fmt.Errorf("no manifest found for %q - try 'sous init' first", mid)
			logging.Deliver(newUpdateErrorMessage(tries, sid, did, user, start, err), ls)
			return sous.NewDeployments(), err
		}

		tryLimit = len(manifest.Deployments)

		if gdm, err = state.Deployments(); err != nil {
			logging.Deliver(newUpdateErrorMessage(tries, sid, did, user, start, err), ls)
			return sous.NewDeployments(), err
		}

		if err := updateState(state, gdm, sid, did); err != nil {
			logging.Deliver(newUpdateErrorMessage(tries, sid, did, user, start, err), ls)
			return sous.NewDeployments(), err
		}

		if err := sm.WriteState(state, user); err != nil {
			if !restful.Retryable(err) {
				logging.Deliver(newUpdateErrorMessage(tries, sid, did, user, start, err), ls)
				return sous.NewDeployments(), err
			}

			continue
		}

		logging.Deliver(newUpdateSuccessMessage(tries, sid, did, manifest, user, start), ls)
		return gdm, nil
	}

	err := errors.Errorf("tried %d to update %v - %v", tryLimit, sid, did)
	logging.Deliver(newUpdateErrorMessage(tryLimit, sid, did, user, start, err), ls)
	return sous.NewDeployments(), err
}

func updateState(s *sous.State, gdm sous.Deployments, sid sous.SourceID, did sous.DeploymentID) error {
	deployment, ok := gdm.Get(did)
	if !ok {
		deployment = &sous.Deployment{}
	}

	deployment.SourceID = sid
	deployment.ClusterName = did.Cluster

	return s.UpdateDeployments(deployment)
}
