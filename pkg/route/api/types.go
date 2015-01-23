package api

import (
	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
)

// Route encapsulates the inputs needed to connect an alias to endpoints.
type Route struct {
	kapi.TypeMeta   `json:",inline"`
	kapi.ObjectMeta `json:"metadata,omitempty"`

	// Required: Alias/DNS that points to the service
	// Can be host or host:port
	// host and port are combined to follow the net/url URL struct
	Host string `json:"host"`
	// Optional: Path that the router watches for, to route traffic for to the service
	Path string `json:"path,omitempty"`

	// the name of the service that this route points to
	ServiceName string `json:"serviceName"`

	//TLS provides the ability to configure certificates and termination for the route
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`
}

// RouteList is a collection of Routes.
type RouteList struct {
	kapi.TypeMeta `json:",inline"`
	kapi.ListMeta `json:"metadata,omitempty"`
	Items         []Route `json:"items"`
}

// Configuration used to secure a route and provide termination
type TLSConfig struct {
	//The termination type, if empty default will be edge
	Termination TLSTerminationType `json:"termination,omitempty" yaml:"termination,omitempty"`

	//certificate contents
	Certificate string `json:"certificate,omitempty" yaml:"certificate,omitempty"`

	//key file contents.  Required for edge termination.
	Key string `json:"key,omitempty" yaml:"key,omitempty"`

	//CA Certificate contents
	CACertificate string `json:"caCertificate,omitempty" yaml:"caCertificate,omitempty"`

	//when using reencrypt termination this file should be provided in order to have routers use it for health checks
	//on the secure connection
	DestinationCACertificate string `json:"destinationCACertificate,omitempty" yaml:"destinationCACertificate,omitempty"`
}

// dictates where the secure communication will stop
type TLSTerminationType string

const (
	// terminate encryption at the edge router
	TLSTerminationEdge TLSTerminationType = "edge"
	// terminate encryption at the destination, the destination is responsible for decrypting traffic
	TLSTerminationPassthrough TLSTerminationType = "passthrough"
	// terminate encryption at the edge router and re-encrypt it with a new certificate supplied by the pod
	TLSTerminationReencrypt TLSTerminationType = "reencrypt"
)
