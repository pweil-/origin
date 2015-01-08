package templaterouter

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"text/template"

	"github.com/golang/glog"

	routeapi "github.com/openshift/origin/pkg/route/api"
)

const (
	ProtocolHTTP  = "http"
	ProtocolHTTPS = "https"
	ProtocolTLS   = "tls"
)

const (
	RouteFile = "/var/lib/containers/router/routes.json"
	CertDir   = "/var/lib/containers/router/certs/"
)

// templateRouter is a backend-agnostic router implementation
// that generates configuration files via a set of templates
// and manages the backend process with a reload script.
type templateRouter struct {
	templates        map[string]*template.Template
	reloadScriptPath string
	state            map[string]ServiceUnit
	certManager		 certManager
}

func newTemplateRouter(templates map[string]*template.Template, reloadScriptPath string) (*templateRouter, error) {
	router := &templateRouter{templates, reloadScriptPath, map[string]ServiceUnit{}, certManager{}}
	err := router.readState()
	return router, err
}

func (r *templateRouter) readState() error {
	dat, err := ioutil.ReadFile(RouteFile)
	// XXX: rework
	if err != nil {
		r.state = make(map[string]ServiceUnit)
		return nil
	}

	return json.Unmarshal(dat, &r.state)
}

// Commit refreshes the backend and persists the router state.
func (r *templateRouter) Commit() error {
	glog.V(4).Info("Commiting router changes")

	var err error
	if err = r.writeState(); err != nil {
		return err
	}

	if r.writeConfig(); err != nil {
		return err
	}

	if r.reloadRouter(); err != nil {
		return err
	}

	return nil
}

// writeState writes the state of this router to disk.
func (r *templateRouter) writeState() error {
	dat, err := json.MarshalIndent(r.state, "", "  ")
	if err != nil {
		glog.Errorf("Failed to marshal route table: %v", err)
		return err
	}
	err = ioutil.WriteFile(RouteFile, dat, 0644)
	if err != nil {
		glog.Errorf("Failed to write route table: %v", err)
		return err
	}

	return nil
}

// write the config to disk
func (r *templateRouter) writeConfig() error {
	//write out any certificate files that don't exist
	//todo: better way so this doesn't need to create lots of files every time state is written, probably too expensive
	for _, serviceUnit := range r.state {
		for _, cfg := range serviceUnit.ServiceAliasConfigs {
			r.certManager.writeCertificatesForConfig(&cfg)
		}
	}

	for path, template := range r.templates {
		file, err := os.Create(path)
		if err != nil {
			glog.Errorf("Error creating config file %v: %v", path, err)
			return err
		}

		err = template.Execute(file, r.state)
		if err != nil {
			glog.Errorf("Error executing template for file %v: %v", path, err)
			return err
		}

		file.Close()
	}

	return nil
}


// reloadRouter executes the router's reload script.
func (r *templateRouter) reloadRouter() error {
	cmd := exec.Command(r.reloadScriptPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Error reloading router: %v\n Reload output: %v", err, string(out))
	}
	return err
}

// CreateFrontend creates a new frontend named with the given id.
func (r *templateRouter) CreateServiceUnit(id string) {
	frontend := ServiceUnit{
		Name:          id,
		ServiceAliasConfigs:      make(map[string]ServiceAliasConfig),
		EndpointTable: make(map[string]Endpoint),
	}

	r.state[id] = frontend
}

// FindServiceUnit finds the frontend with the given id.
func (r *templateRouter) FindServiceUnit(id string) (v ServiceUnit, ok bool) {
	v, ok = r.state[id]
	return
}

// DeleteFrontend deletes the frontend with the given id.
func (r *templateRouter) DeleteServiceUnit(id string) {
	delete(r.state, id)
}

// DeleteEndpoints deletes the endpoints for the frontend with the given id.
func (r *templateRouter) DeleteEndpoints(id string) {
	frontend, ok := r.FindServiceUnit(id)
	if !ok {
		return
	}
	frontend.EndpointTable = make(map[string]Endpoint)

	r.state[id] = frontend
}

func (r *templateRouter) routeKey(route *routeapi.Route) string{
	return route.Host + "-" + route.Path
}

// AddRoute adds a route for the given id
func (r *templateRouter) AddRoute(id string, route *routeapi.Route) {
	frontend, _ := r.FindServiceUnit(id)

	backendKey := r.routeKey(route)

	config := ServiceAliasConfig {
		Host: route.Host,
		Path: route.Path,
	}

	if len(route.TLS.Termination) > 0 {
		config.TLSTermination = route.TLS.Termination

		if route.TLS.Termination != routeapi.TLSTerminationPod {
			if config.Certificates == nil {
				config.Certificates = make(map[string]Certificate)
			}

			cert := Certificate{
				ID: route.Host,
				Contents: []byte(route.TLS.Certificate),
				PrivateKey: []byte(route.TLS.Key),
				PrivateKeyPassword: route.TLS.KeyPassPhrase,
			}

			config.Certificates[cert.ID] = cert

			if len(route.TLS.CACertificate) > 0 {
				caCert := Certificate {
					ID: route.Host + "_ca",
					Contents: []byte(route.TLS.CACertificate),
				}

				config.Certificates[cert.ID] = caCert
			}
			//todo: re-encrypt certs
		}
	}

	//create or replace
	frontend.ServiceAliasConfigs[backendKey] = config
	r.state[id] = frontend
}

// RemoveAlias removes the given alias for the given id.
func (r *templateRouter) RemoveRoute(id string, route *routeapi.Route) {
	_, ok := r.state[id]

	if !ok {
		return
	}

	delete(r.state[id].ServiceAliasConfigs, r.routeKey(route))
}

// AddRoute adds new Endpoints for the given id.
func (r *templateRouter) AddEndpoints(id string, endpoints []Endpoint) {
	frontend, _ := r.FindServiceUnit(id)

	//only add if it doesn't already exist
	for _, ep := range endpoints {
		if _, ok := frontend.EndpointTable[ep.ID]; !ok {
			newEndpoint := Endpoint {ep.ID, ep.IP, ep.Port}
			frontend.EndpointTable[ep.ID] = newEndpoint
		}
	}

	r.state[id] = frontend
}

func cmpStrSlices(first []string, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for _, fi := range first {
		found := false
		for _, si := range second {
			if fi == si {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
