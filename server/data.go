package server

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sort"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type (
	// ResponseMeta contains metadata to include in API response bodies.
	ResponseMeta struct {
		// Links is a set of links related to a response body.
		Links map[string]string
	}

	// NameData structs contain the pair of clustername to URL for data transfer
	NameData struct {
		ClusterName string
		URL         string
	}

	// ServerListData is the DTO for lists of servers
	ServerListData struct { // not actually a stutter - "server" means two different things.
		Servers []NameData
	}

	// GDMWrapper is the DTO wrapper for sous.Deployments
	GDMWrapper struct {
		Deployments []*sous.Deployment
	}

	// ClientUser is a local alias for sous.User
	ClientUser sous.User

	// StateManager is a DI adapter
	StateManager struct {
		sous.StateManager
	}

	// DeploymentQueuesResponse is used by the Deployment queue handler
	DeploymentQueuesResponse struct {
		Queues map[string]QueueDesc
	}

	// QueueDesc describes the queue related to a Deployment
	QueueDesc struct {
		sous.DeploymentID
		Length int
	}

	// SingleDeploymentBody is the response struct returned from handlers
	// of HTTP methods of a SingleDeploymentResource.
	SingleDeploymentBody struct {
		Meta       ResponseMeta
		Deployment sous.Deployment
	}
)

// EmptyReceiver implements Comparable on ServerListData
func (ld *ServerListData) EmptyReceiver() restful.Comparable {
	return &ServerListData{Servers: []NameData{}}
}

// VariancesFrom implements Comparable on ServerListData
func (ld *ServerListData) VariancesFrom(other restful.Comparable) restful.Variances {
	switch ol := other.(type) {
	default:
		return restful.Variances{"not a list of Deployments"}
	case *ServerListData:
		if len(ld.Servers) != len(ol.Servers) {
			return restful.Variances{"server list lengths differ"}
		}
		for _, l := range ld.Servers {
			var found *NameData
			for _, r := range ol.Servers {
				if l.ClusterName == r.ClusterName && l.URL == r.URL {
					found = &r
					break
				}
			}
			if found == nil {
				return restful.Variances{"No match found for " + l.ClusterName}
			}
		}
		return restful.Variances{}
	}
}

// EmptyReceiver implements Comparable on gdmWrapper
func (g *GDMWrapper) EmptyReceiver() restful.Comparable {
	return &GDMWrapper{Deployments: []*sous.Deployment{}}
}

// VariancesFrom implements Comparable on gdmWrapper
func (g *GDMWrapper) VariancesFrom(other restful.Comparable) restful.Variances {
	switch og := other.(type) {
	default:
		return restful.Variances{"Not a gdmWrapper"}
	case *GDMWrapper:
		return g.unwrap().VariancesFrom(og.unwrap())
	}
}

// AddHeaders implements HeaderAdder on GDMWrapper
// GDMWrappers add an Etag to the response
func (g GDMWrapper) AddHeaders(headers http.Header) {
	headers.Add("Etag", g.etag())
}

// Etag returns a string suitable for use in an Etag header for this data type.
// n.b. that Etags are generated automatically for most restful bodies
// GDMWrapper is unique because any *set* of Deployments is equivalent, regardless of their order.
func (g GDMWrapper) etag() string {
	deps := make([]*sous.Deployment, 0, len(g.Deployments))
	copy(deps, g.Deployments)
	sort.Slice(deps, func(i, j int) bool { return deps[i].ID().String() < deps[j].ID().String() })

	hash := sha512.New()
	for _, dep := range deps {
		hash.Write([]byte(dep.String()))
	}

	return "w/" + base64.URLEncoding.EncodeToString(hash.Sum(nil))
}

func (g *GDMWrapper) unwrap() *sous.Deployments {
	ds := sous.NewDeployments(g.Deployments...)
	return &ds
}

// AddHeaders implements HeaderAdder on SingleDeploymentBody
func (b SingleDeploymentBody) AddHeaders(headers http.Header) {
	headers.Add("Etag", b.etag())
	queuedURL, ok := b.Meta.Links["queuedDeployAction"]
	if ok {
		headers.Add("Location", queuedURL)
	}
}

// EmptyReceiver implements Comparable on SingleDeploymentBody
func (b SingleDeploymentBody) EmptyReceiver() restful.Comparable {
	return &SingleDeploymentBody{}
}

// VariancesFrom implements Comparable on SingleDeploymentBody
func (b SingleDeploymentBody) VariancesFrom(other restful.Comparable) restful.Variances {
	switch ob := other.(type) {
	default:
		return restful.Variances{"Not a SingleDeploymentBody"}
	case *SingleDeploymentBody:
		_, diffs := (&b.Deployment).Diff(&ob.Deployment)
		return restful.Variances(diffs)
	}
}

// Etag returns a string suitable for use in an Etag header for this data type.
// SingleDeploymentBody includes a Meta subobject, whose values may vary
// independantly of the Etag.
func (b *SingleDeploymentBody) etag() string {
	hash := sha512.New()
	ds, err := json.Marshal(b.Deployment)
	if err != nil {
		panic("unmarshallable SingleDeploymentBody.DeploySpec")
	}

	hash.Write(ds)
	return "w/" + base64.URLEncoding.EncodeToString(hash.Sum(nil))
}
