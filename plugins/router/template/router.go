package templaterouter

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"text/template"
	"time"

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
	state            map[string]Frontend
	certManager		 certManager
}

func newTemplateRouter(templates map[string]*template.Template, reloadScriptPath string) (*templateRouter, error) {
	router := &templateRouter{templates, reloadScriptPath, map[string]Frontend{}, certManager{}}
	err := router.readState()
	return router, err
}

func (r *templateRouter) readState() error {
	dat, err := ioutil.ReadFile(RouteFile)
	// XXX: rework
	if err != nil {
		r.state = make(map[string]Frontend)
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

// writeConfig processes the templates and writes config files.
func (r *templateRouter) writeConfig() error {
	//write out any certificate files that don't exist
	//todo: better way so this doesn't need to create lots of files every time state is written, probably too expensive
	for _, fe := range r.state {
		for _, be := range fe.Backends {
			r.certManager.writeCertificatesForBackend(&be)
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
func (r *templateRouter) CreateFrontend(id string, url string) {
	frontend := Frontend{
		Name:          id,
		Backends:      make(map[string]Backend),
		EndpointTable: make(map[string]Endpoint),
		HostAliases:   make([]string, 0),
	}

	if url != "" {
		frontend.HostAliases = append(frontend.HostAliases, url)
	}
	r.state[id] = frontend
}

// FindFrontend finds the frontend with the given id.
func (r *templateRouter) FindFrontend(id string) (v Frontend, ok bool) {
	v, ok = r.state[id]
	return
}

// DeleteFrontend deletes the frontend with the given id.
func (r *templateRouter) DeleteFrontend(id string) {
	delete(r.state, id)
}

// DeleteBackends deletes the backends for the frontend with the given id.
func (r *templateRouter) DeleteBackends(id string) {
	frontend, ok := r.state[id]
	if !ok {
		return
	}
	frontend.Backends = make(map[string]Backend)
	frontend.EndpointTable = make(map[string]Endpoint)

	r.state[id] = frontend
}

// AddAlias adds a host alias for the given id.
func (r *templateRouter) AddAlias(id, alias string) {
	frontend := r.state[id]
	for _, v := range frontend.HostAliases {
		if v == alias {
			return
		}
	}

	frontend.HostAliases = append(frontend.HostAliases, alias)
	r.state[id] = frontend
}

func (r *templateRouter) SecureRoute(id string, route *routeapi.Route) {
	if &route.TLS != nil {
		//get the frontend for the route
		fe := r.state[id]

		//find the matching backend (check for the path or take the empty one)
		for idx, be := range fe.Backends {

			if be.FePath == route.Path {
				if be.Certificates == nil {
					be.Certificates = make(map[string]Certificate)
				}

				be.TLSTermination = route.TLS.Termination

				cert := Certificate{
					Contents: []byte(route.TLS.Certificate),
					PrivateKey: []byte(route.TLS.Key),
					PrivateKeyPassword: route.TLS.KeyPassPhrase,
				}

				be.Certificates[route.Host] = cert

				if len(route.TLS.CACertificate) > 0 {
					caCert := Certificate {
						Contents: []byte(route.TLS.CACertificate),
					}

					be.Certificates[route.Host + "_ca"] = caCert
				}
				//todo: re-encrypt certs
				r.state[id].Backends[idx] = be
				return
			}
		}

		//If we got here it means we tried securing a route that didn't have a matching backend
		glog.Warningf("Tried to secure route %s but couldn't find matching backend with path %s", id, route.Path)
	}
}

// RemoveAlias removes the given alias for the given id.
func (r *templateRouter) RemoveAlias(id, alias string) {
	frontend := r.state[id]
	newAliases := []string{}
	for _, v := range frontend.HostAliases {
		if v == alias || v == "" {
			continue
		}
		newAliases = append(newAliases, v)
	}

	frontend.HostAliases = newAliases
	r.state[id] = frontend
}

// AddRoute adds new Endpoints for the given id.
func (r *templateRouter) AddRoute(id string, back *Backend, endpoints []Endpoint) {
	frontend := r.state[id]

	// determine which endpoints from the input are new
	newEndpoints := make([]string, 1)
	for _, input := range endpoints {
		if input.IP == "" || input.Port == "" {
			continue
		}

		found := false
		for _, ep := range frontend.EndpointTable {
			if ep.IP == input.IP && ep.Port == input.Port {
				newEndpoints = append(newEndpoints, ep.ID)
				found = true
				break
			}
		}

		if !found {
			endpointID := makeID()
			ep := Endpoint{endpointID, input.IP, input.Port}
			frontend.EndpointTable[endpointID] = ep
			newEndpoints = append(newEndpoints, ep.ID)
		}
	}

	// locate a backend that may already exist with this protocol and fe/be path
	found := false
	for _, be := range frontend.Backends {
		if be.FePath == back.FePath && be.BePath == back.BePath && cmpStrSlices(back.Protocols, be.Protocols) {
			for _, epID := range newEndpoints {
				be.EndpointIDs = append(be.EndpointIDs, epID)
			}
			frontend.Backends[be.ID] = be
			found = true
			break
		}
	}

	// create a new backend if none was found.
	if !found {
		backendID := makeID()
		frontend.Backends[backendID] = Backend{backendID,
			back.FePath,
			back.BePath,
			back.Protocols,
			newEndpoints,
			routeapi.TLSTerminationEdge,
			nil}
	}

	r.state[id] = frontend
}

// TODO: make a better ID generator
func makeID() string {
	var s string
	s = strconv.FormatInt(time.Now().UnixNano(), 16)
	return s
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
