package policy

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"net/http"

	"k8s.io/kubernetes/pkg/apis/imagepolicy/v1alpha1"
)

type ImagePolicyServer struct {
	cert       []byte
	key        []byte
	caCert     []byte
	listenAddr string
}

func NewImagePolicyServer(cert, key, caCert []byte, listenAddr string) *ImagePolicyServer {
	return &ImagePolicyServer{
		listenAddr: listenAddr,
		cert:       cert,
		key:        key,
		caCert:     caCert,
	}
}

func (s *ImagePolicyServer) HandleSignatureRequest(w http.ResponseWriter, r *http.Request) {
	// TODO real decode/encode, handling of the request
	var review v1alpha1.ImageReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode body: %v", err), http.StatusBadRequest)
		return
	}

	review.Status = v1alpha1.ImageReviewStatus{
		Allowed: true,
		Reason:  "default",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(review)
}

func (s *ImagePolicyServer) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc(("/healthz"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	mux.HandleFunc(("/policy"), s.HandleSignatureRequest)

	tlsCert, err := tls.X509KeyPair(s.cert, s.key)
	if err != nil {
		return err
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}

	if s.caCert != nil {
		rootCAs := x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(s.caCert)
		cfg.ClientCAs = rootCAs
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	listener, err := tls.Listen("tcp", s.listenAddr, cfg)
	if err != nil {
		return err
	}
	glog.Infof("Listening on %s", s.listenAddr)

	if err := http.Serve(listener, mux); err != nil {
		return err
	}
	return nil
}
