package templaterouter

import (
	routeapi "github.com/openshift/origin/pkg/route/api"
)

// A frontend is a representation of a service along with the corresponding backends that
// support the service and the unique endpoints that implement the service
type ServiceUnit struct {
	// Corresponds to a service name & namespace.  Uniquely identifies the frontend
	Name          string
	// Endpoints that back the service, this translates into a final backend implementation for routers
	// keyed by IP:port for easy access
	EndpointTable map[string]Endpoint
	// Collection of unique backends that support this service, keyed by host + path
	ServiceAliasConfigs      map[string]ServiceAliasConfig
}

// A backend is used by the router to realize the configuration.  A backend is driven by
// a route object 1:1 and is uniquely identified by host + path
type ServiceAliasConfig struct {
	// Required host name ie www.example.com
	Host string
	// An optional path.  Ie. www.example.com/myservice where "myservice" is the path
	Path string
	// Termination policy for this backend, drives the mapping files and router configuration
	TLSTermination routeapi.TLSTerminationType
	// Certificates used for securing this backend.  Keyed by the cert id
	Certificates map[string]Certificate
}

type Certificate struct {
	ID                 string
	Contents           []byte
	PrivateKey         []byte
	PrivateKeyPassword string
}

type Endpoint struct {
	ID   string
	IP   string
	Port string
}
