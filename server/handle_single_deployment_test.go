package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/nyarly/spies"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
	"github.com/samsalisbury/semv"
)

func TestSingleDeploymentResource(t *testing.T) {
	dm, _ := sous.NewDeploymentManagerSpy()
	qs, _ := sous.NewQueueSetSpy()
	cl := ComponentLocator{
		DeploymentManager: dm,
		QueueSet:          qs,
	}
	r := newSingleDeploymentResource(cl)

	rm := routemap(cl)

	rw := httptest.NewRecorder()

	t.Run("Get()", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://sous.example.com/single-deployment", nil)

		gex := r.Get(rm, rw, req, nil)

		if gex == nil {
			t.Fatalf("r.Get returned nil")
		}

		gsdh, is := gex.(*GETSingleDeploymentHandler)
		if !is {
			t.Fatalf("r.GET did not return a GETSingleDeploymentHandler")
		}
		if gsdh.responseWriter != rw {
			t.Errorf("GET handler didn't get the ResponseWriter")
		}
		if gsdh.req != req {
			t.Errorf("GET handler didn't get the Request")
		}
		if gsdh.DeploymentManager != cl.DeploymentManager {
			t.Errorf("GET handler didn't get the DeploymentManager")
		}
	})

	t.Run("Put()", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "http://sous.example.com/single-deployment", bytes.NewBufferString("{}"))
		pex := r.Put(rm, rw, req, nil)
		if pex == nil {
			t.Fatalf("r.Put returned nil")
		}

		psdh, is := pex.(*PUTSingleDeploymentHandler)
		if !is {
			t.Fatalf("r.Put did not return a PUTSingleDeploymentHandler")
		}
		if psdh.responseWriter != rw {
			t.Errorf("PUT handler didn't get the ResponseWriter")
		}
		if psdh.req != req {
			t.Errorf("PUT handler didn't get the Request")
		}
		if psdh.DeploymentManager != cl.DeploymentManager {
			t.Errorf("PUT handler didn't get the DeploymentManager")
		}
		if psdh.QueueSet != cl.QueueSet {
			t.Errorf("PUT handler didn't get the QueueSet")
		}
		if psdh.routeMap != rm {
			t.Errorf("PUT handler didn't get the route map")
		}
	})

}

type psdhExScenario struct {
	handler           restful.Exchanger
	response          interface{}
	status            int
	deploymentManager *spies.Spy
	queueSet          *spies.Spy
}

func (scn *psdhExScenario) hasDeployment(dep *sous.Deployment) {
	scn.deploymentManager.MatchMethod("ReadDeployment", spies.AnyArgs, dep, nil)
}

func (scn *psdhExScenario) exercise() {
	scn.response, scn.status = scn.handler.Exchange()
}

func (scn psdhExScenario) assertStatus(t *testing.T, expected int) {
	t.Helper()
	if scn.status != expected {
		t.Errorf("Expected status %d, got %d", expected, scn.status)
	}
}

func (scn psdhExScenario) assertHeader(t *testing.T, wantKey, wantValue string) {
	t.Helper()

	getHeader, ok := scn.response.(restful.HeaderAdder)
	if !ok {
		t.Errorf("no header")
		return
	}
	h := http.Header{}
	getHeader.AddHeaders(h)

	gotValue := h.Get(wantKey)
	if gotValue != wantValue {
		t.Errorf("got:\n%q=%q\nwant:\n%q=%q", wantKey, gotValue, wantKey, wantValue)
	}
}

func (scn psdhExScenario) assertStringBody(t *testing.T, expected string) {
	t.Helper()
	body, is := scn.response.(string)
	if !is {
		t.Errorf("Expected a simple string response - got %T", scn.response)
		return
	}
	if !strings.Contains(body, expected) {
		t.Errorf("Expected response to contain %q, but not found in %q", expected, body)
	}
}

func (scn psdhExScenario) assertDeploymentWritten(t *testing.T) {
	t.Helper()
	calls := scn.deploymentManager.CallsTo("WriteDeployment")
	if len(calls) == 0 {
		t.Errorf("Expected that a deployment would be written, but none were.")
	}
}

func (scn psdhExScenario) assertR11nQueued(t *testing.T) {
	t.Helper()
	calls := scn.queueSet.CallsTo("Push")
	if len(calls) == 0 {
		t.Errorf("Expected that a recitification would be queued, but none were.")
	}
}

func (scn psdhExScenario) assertNoR11nQueued(t *testing.T) {
	t.Helper()
	piecalls := scn.queueSet.CallsTo("PushIfEmpty")
	pcalls := scn.queueSet.CallsTo("Push")
	if len(piecalls) != 0 || len(pcalls) != 0 {
		t.Errorf("Expected that no recitification would be queued, but one was.")
	}
}

func TestPUTSingleDeploymentHandler_Exchange(t *testing.T) {
	setup := func(sent *SingleDeploymentBody, did map[string]string) *psdhExScenario {
		// Setup

		dmSpy, dmCtrl := sous.NewDeploymentManagerSpy()
		qs, qsCtrl := sous.NewQueueSetSpy()
		cl := ComponentLocator{
			DeploymentManager: dmSpy,
			QueueSet:          qs,
		}
		r := newSingleDeploymentResource(cl)

		rm := routemap(cl)

		values := url.Values{}
		for k, v := range did {
			values.Add(k, v)
		}
		url, err := url.Parse("http://sous.example.com/single-deployment?" + values.Encode())
		if err != nil {
			t.Fatalf("Error parsing URL during setup: %v", err)
		}

		bs, err := json.Marshal(sent)
		if err != nil {
			t.Fatalf("Error marshalling test sent body: %v", err)
		}
		body := bytes.NewBuffer(bs)

		req := httptest.NewRequest("PUT", url.String(), body)
		req.Header.Set("Sous-User-Name", "Test User")
		req.Header.Set("Sous-User-Email", "testuser@example")

		rw := httptest.NewRecorder()

		psd := r.Put(rm, rw, req, nil)

		return &psdhExScenario{
			handler:           psd,
			deploymentManager: dmCtrl,
			queueSet:          qsCtrl,
		}
	}

	didQuery := func(repo, offset, cluster, flavor string) map[string]string {
		return map[string]string{
			"repo":    repo,
			"offset":  offset,
			"cluster": cluster,
			"flavor":  flavor,
		}
	}

	t.Run("query parsing error", func(t *testing.T) {
		scenario := setup(nil, map[string]string{})
		scenario.hasDeployment(sous.DeploymentFixture(""))
		scenario.exercise()

		scenario.assertStatus(t, 400)
		scenario.assertStringBody(t, `Cannot decode Deployment ID:`)
	})

	t.Run("body parsing error", func(t *testing.T) {
		scenario := setup(nil, didQuery("github.com/opentable/something", "", "cluster", ""))
		scenario.hasDeployment(sous.DeploymentFixture(""))
		scenario.exercise()

		scenario.assertStatus(t, 400)
		scenario.assertStringBody(t, `Invalid deployment`)
	})

	t.Run("no matching deployment", func(t *testing.T) {
		dep := sous.DeploymentFixture("")
		scenario := setup(&SingleDeploymentBody{Deployment: *dep}, didQuery("github.com/user1/repo1", "", "cluster1", ""))
		scenario.deploymentManager.MatchMethod("ReadDeployment", spies.AnyArgs, nil, errors.New("no deployment found"))
		scenario.exercise()

		scenario.assertStatus(t, 404)
		scenario.assertStringBody(t, "No deployment with ID")
	})

	t.Run("no change necessary", func(t *testing.T) {
		dep := sous.DeploymentFixture("")
		scenario := setup(&SingleDeploymentBody{Deployment: *dep}, didQuery("github.com/user1/repo1", "", "cluster1", ""))
		scenario.hasDeployment(sous.DeploymentFixture(""))
		scenario.exercise()

		scenario.assertNoR11nQueued(t)
		scenario.assertStatus(t, 200)
	})

	t.Run("change version", func(t *testing.T) {
		dep := sous.DeploymentFixture("")
		body := &SingleDeploymentBody{Deployment: *dep}
		body.Deployment.SourceID.Version = semv.MustParse("2.0.0")
		query := didQuery("github.com/user1/repo1", "", "cluster1", "")
		scenario := setup(body, query)
		scenario.hasDeployment(sous.DeploymentFixture(""))
		qr := &sous.QueuedR11n{
			ID: "actionid1",
		}
		scenario.queueSet.MatchMethod("Push", spies.AnyArgs, qr, true)
		scenario.exercise()

		scenario.assertStatus(t, 201)
		scenario.assertDeploymentWritten(t)
		scenario.assertR11nQueued(t)
		scenario.assertHeader(t, "Location",
			"/deploy-queue-item?action=actionid1&cluster=cluster1&flavor=&offset=&repo=github.com%2Fuser1%2Frepo1")
	})

	t.Run("WriteDeployment error", func(t *testing.T) {
		dep := sous.DeploymentFixture("")
		dep.NumInstances = 7
		scenario := setup(&SingleDeploymentBody{Deployment: *dep}, didQuery("github.com/user1/repo1", "", "cluster1", ""))
		scenario.hasDeployment(sous.DeploymentFixture(""))
		scenario.deploymentManager.MatchMethod("WriteDeployment", spies.AnyArgs, errors.New("an error occurred"))
		scenario.exercise()

		scenario.assertDeploymentWritten(t)
		scenario.assertStatus(t, 500)
		scenario.assertStringBody(t, "Failed to write deployment")
	})

	t.Run("PushToQueueSet error", func(t *testing.T) {
		dep := sous.DeploymentFixture("")
		dep.NumInstances = 7
		scenario := setup(&SingleDeploymentBody{Deployment: *dep}, didQuery("github.com/user1/repo1", "", "cluster1", ""))
		scenario.hasDeployment(sous.DeploymentFixture(""))
		scenario.queueSet.MatchMethod("Push", spies.AnyArgs, &sous.QueuedR11n{}, false)
		scenario.exercise()

		scenario.assertDeploymentWritten(t)
		scenario.assertStatus(t, 409)
		scenario.assertStringBody(t, "Queue full, please try again later.")
	})
}
