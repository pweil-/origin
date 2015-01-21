// +build integration,!no-docker,docker
package router

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"io"
)

func NewTestHttpServer() *TestHttpServer {
	endpointChannel := make(chan string)
	routeChannel := make(chan string)

	return &TestHttpServer{
		MasterHttpAddr: "0.0.0.0:8080",
		PodHttpAddr:    "0.0.0.0:8888",
		PodHttpsAddr:   "0.0.0.0:8443",
		PodHttpsCert:   []byte(Example2Cert),
		PodHttpsKey:    []byte(Example2Key),
		PodHttpsCaCert: []byte(ExampleCACert),
		EndpointChannel: endpointChannel,
		RouteChannel: routeChannel,
	}
}

type TestHttpServer struct {
	MasterHttpAddr string
	PodHttpAddr    string
	PodHttpsAddr   string
	PodHttpsCert   []byte
	PodHttpsKey    []byte
	PodHttpsCaCert []byte
	EndpointChannel chan string
	RouteChannel chan string

	listeners []net.Listener
}

const (
	HelloMaster = "Hello OpenShift!"
	HelloPod = "Hello Pod!"
	HelloPodSecure = "Hello Pod Secure!"
)

func (s *TestHttpServer) handleHelloMaster(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, HelloMaster)
}

func (s *TestHttpServer) handleHelloPod(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, HelloPod)
}

func (s *TestHttpServer) handleHelloPodSecure(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, HelloPodSecure)
}

func (s *TestHttpServer) handleRouteWatch(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, <- s.RouteChannel)
}

func (s *TestHttpServer) handleRouteList(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "{}")
}

func (s *TestHttpServer) handleEndpointWatch(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, <- s.EndpointChannel)
}

func (s *TestHttpServer) handleEndpointList(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "{}")
}

func (s *TestHttpServer) Stop() {
	if s.listeners != nil && len(s.listeners) > 0 {
		for _, l := range s.listeners {
			if l != nil {
				fmt.Printf("Stopping listener at %s\n", l.Addr().String())
				l.Close()
			}
		}
	}
}

func (s *TestHttpServer) Start() error {
    s.listeners = make([]net.Listener, 3)

	masterServer := http.NewServeMux()
	masterServer.HandleFunc("/api/v1beta1/endpoints", s.handleEndpointList)
	masterServer.HandleFunc("/api/v1beta1/watch/endpoints", s.handleEndpointWatch)

	masterServer.HandleFunc("/osapi/v1beta1/routes", s.handleRouteList)
	masterServer.HandleFunc("/osapi/v1beta1/watch/routes", s.handleRouteWatch)

	masterServer.HandleFunc("/", s.handleHelloMaster)

	if err := s.startServing(s.MasterHttpAddr, masterServer); err != nil {
		return err
	}

	unsecurePodServer := http.NewServeMux()
	unsecurePodServer.HandleFunc("/", s.handleHelloPod)
	if err := s.startServing(s.PodHttpAddr, unsecurePodServer); err != nil {
		return err
	}

	securePodServer := http.NewServeMux()
	securePodServer.HandleFunc("/", s.handleHelloPodSecure)
	if err := s.startServingTLS(s.PodHttpsAddr, s.PodHttpsCert, s.PodHttpsKey, s.PodHttpsCaCert, securePodServer); err != nil {
		return err
	}

	return nil
}

func (s *TestHttpServer) startServing(addr string, handler *http.ServeMux) error {
	listener, err := net.Listen("tcp", addr)

	if err != nil {
		return err
	}

	s.listeners = append(s.listeners, listener)

	fmt.Printf("Started, serving at %s\n", listener.Addr().String())

	go func() {
		err := http.Serve(listener, handler)

		if err != nil {
			fmt.Printf("Server message: %v", err)
			s.Stop()
		}
	}()

	return nil
}

func (s *TestHttpServer) startServingTLS(addr string, cert []byte, key []byte, caCert []byte, handler *http.ServeMux) error {
	tlsCert, err := tls.X509KeyPair(append(cert, caCert...), key)

	if err != nil {
		return err
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}

	listener, err := tls.Listen("tcp", addr, cfg)

	if err != nil {
		return err
	}

	s.listeners = append(s.listeners, listener)
	fmt.Printf("Started, serving TLS at %s\n", listener.Addr().String())

	go func() {
		err := http.Serve(listener, handler)

		if err != nil {
			fmt.Printf("Server message: %v", err)
		}
	}()

	return nil
}
