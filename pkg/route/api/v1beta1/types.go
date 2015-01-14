package v1beta1

import (
	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
)

// Route encapsulates the inputs needed to connect an alias to endpoints.
type Route struct {
	kapi.TypeMeta   `json:",inline" yaml:",inline"`
	kapi.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Required: Alias/DNS that points to the service
	// Can be host or host:port
	// host and port are combined to follow the net/url URL struct
	Host string `json:"host" yaml:"host"`
	// Optional: Path that the router watches for, to route traffic for to the service
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// the name of the service that this route points to
	ServiceName string `json:"serviceName" yaml:"serviceName"`

	//TLS provides the ability to configure certificates and termination for the route
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`
}

// RouteList is a collection of Routes.
type RouteList struct {
	kapi.TypeMeta `json:",inline" yaml:",inline"`
	kapi.ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items         []Route `json:"items" yaml:"items"`
}

// Configuration used to secure a route and provide termination
type TLSConfig struct {
	//The termination type, if empty default will be edge
	Termination TLSTerminationType `json:"termination,omitempty" yaml:"termination,omitempty"`

	//used during Route configuration to specify the file to upload.  Required for edge termination
	CertificateFile string `json:"certificateFile,omitempty" yaml:"certificateFile,omitempty"`
	//if specified Certificate contains the contents of CertificateFile.  Otherwise, CertificateFile
	//will be parsed into this for storage
	Certificate []byte `json:"certificate,omitempty" yaml:"certificate,omitempty"`

	//used during Route configuration to specify the file to upload.  Required for edge termination
	KeyFile string `json:"keyFile,omitempty" yaml:"keyFile,omitempty"`
	//if specified Key contains the contents of KeyFile.  Otherwise, KeyFile will be parsed into
	//this for storage
	Key []byte `json:"key,omitempty" yaml:"key,omitempty"`

	//password for the keyfile
	//TODO: currently unsupported
	KeyPassPhrase string `json:"keyPassPhrase,omitempty" yaml:"keyPassPhrase,omitempty"`

	//used during Route configuration to specify the file to upload.  If not specified
	//then it is assumed that the CA chain is concatenated in the certificate file.  If this file is specified
	//it will be added to the end of CertificateFile before parsing
	CACertificateFile string `json:"caCertificateFile,omitempty" yaml:"caCertificateFile,omitempty"`
	//if specified Key contains the contents of CACertificateFile.  Otherwise, CACertificateFile will be parsed into
	//this for storage
	CACertificate []byte `json:"caCertificate,omitempty" yaml:"caCertificate,omitempty"`

	//when using passthrough termination this file can be provided in order to have routers use it for health checks
	//on the secure connection
	DestinationCACertificateFile string `json:"destinationCACertificateFile,omitempty" yaml:"destinationCACertificateFile,omitempty"`
	//holds the parsed DestinationCACertificateFile or can be specified directly
	DestinationCACertificate []byte `json:"destinationCACertificate,omitempty" yaml:"destinationCACertificate,omitempty"`
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
