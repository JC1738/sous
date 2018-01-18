package server

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestHandleDeployments_Exchange(t *testing.T) {

	testCases := []struct {
		dids       []sous.DeploymentID
		wantResult deploymentsResponse
	}{
		{
			dids: []sous.DeploymentID{newDid()},
			wantResult: deploymentsResponse{Deployments: map[sous.DeploymentID]int{
				newDid(): 1,
			}},
		},
	}

	for _, tc := range testCases {
		qs := sous.NewR11nQueueSet()

		for _, did := range tc.dids {
			r11n := &sous.Rectification{
				Pair: sous.DeployablePair{},
			}

			r11n.Pair.SetID(did)

			if _, ok := qs.PushIfEmpty(r11n); !ok {
				t.Fatal("precondition failed: failed to push r11n")
			}

		}

		handler := &GETDeploymentsHandler{
			QueueSet: qs,
		}

		data, gotStatusCode := handler.Exchange()

		const wantStatusCode = 200
		if gotStatusCode != wantStatusCode {
			t.Errorf("got %d; want %d", gotStatusCode, wantStatusCode)
		}

		dr, ok := data.(deploymentsResponse)
		if !ok {
			t.Fatalf("got a %T; want a %T", data, dr)
		}

		wantLen := len(tc.wantResult.Deployments)
		gotLen := len(dr.Deployments)
		if gotLen != wantLen {
			t.Fatalf("got %d queued deployments; want %d", gotLen, wantLen)
		}

		for did, wantCount := range tc.wantResult.Deployments {
			gotCount := dr.Deployments[did]
			if gotCount != wantCount {
				t.Errorf("got %d queued rectifications for %q; want %d", gotCount, did, wantCount)
			}
		}

		testCount := dr.Deployments[sous.DeploymentID{}]
		if testCount != 0 {
			t.Errorf("got %d for empty DeploymentID expected 0", testCount)
		}
	}

}

func newDid() sous.DeploymentID {
	return sous.DeploymentID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: "repo-one",
			},
		},
		Cluster: "cluster-one",
	}
}
