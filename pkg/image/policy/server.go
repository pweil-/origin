package policy

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

type ImagePolicyServer struct {
	cert       []byte
	key        []byte
	caCert     []byte
	listenAddr string
}

func NewImagePolicyServer() *ImagePolicyServer {
	// TODO get cert configuration

	return &ImagePolicyServer{
		listenAddr: "0.0.0.0:443",
	}
}

func (s *ImagePolicyServer) HandleSignatureRequest() {
	// TODO
}

func (s *ImagePolicyServer) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc(("/healthz"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))

	tlsCert, err := tls.X509KeyPair(append(s.cert, s.caCert...), s.key)
	if err != nil {
		return err
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}
	listener, err := tls.Listen("tcp", s.listenAddr, cfg)
	if err != nil {
		return err
	}


	if err := http.Serve(listener, mux); err != nil {
		return err
	}
	return nil
}
