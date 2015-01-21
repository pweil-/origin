package templaterouter

import (
	"strings"
	"text/template"

	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
	"github.com/golang/glog"

	routeapi "github.com/openshift/origin/pkg/route/api"
)

// TemplatePlugin implements the router.Plugin interface to provide
// a template based, backend-agnostic router.
type TemplatePlugin struct {
	Router router
}

type router interface {
	// Mutative operations in this interfance do not return errors.
	// The only error state for these methods is when an unknown
	// frontend key is used; all call sites make certain the frontend
	// is created.

	// CreateFrontend creates a new frontend named with the given id.
	CreateServiceUnit(id string)
	// FindFrontend finds the frontend with the given id.
	FindServiceUnit(id string) (v ServiceUnit, ok bool)

	// AddRoute adds new Endpoints for the given id.
	AddEndpoints(id string, endpoints []Endpoint)
	// DeleteEndpoints deletes the endpoints for the frontend with the given id.
	DeleteEndpoints(id string)

	// AddRoute adds a route for the given id
	AddRoute(id string, route *routeapi.Route)
	// RemoveAlias removes the given alias for the given id.
	RemoveRoute(id string, route *routeapi.Route)

	// Commit refreshes the backend and persists the router state.
	Commit() error
}

// NewTemplatePlugin creates a new TemplatePlugin.
func NewTemplatePlugin(templatePath, reloadScriptPath string) (*TemplatePlugin, error) {
	masterTemplate := template.Must(template.New("config").ParseFiles(templatePath))
	templates := map[string]*template.Template{}

	for _, template := range masterTemplate.Templates() {
		if template == masterTemplate {
			continue
		}

		templates[template.Name()] = template
	}

	router, err := newTemplateRouter(templates, reloadScriptPath)
	return &TemplatePlugin{router}, err
}

// HandleEndpoints processes watch events on the Endpoints resource.
func (p *TemplatePlugin) HandleEndpoints(eventType watch.EventType, endpoints *kapi.Endpoints) error {
	key := endpointsKey(*endpoints)

	glog.V(4).Infof("Processing %d Endpoints for Name: %v (%v)", len(endpoints.Endpoints), endpoints.Name, eventType)

	for i, e := range endpoints.Endpoints {
		glog.V(4).Infof("  Endpoint %d : %s", i, e)
	}

	if _, ok := p.Router.FindServiceUnit(key); !ok {
		p.Router.CreateServiceUnit(key)
	}

	// clear existing endpoints
	p.Router.DeleteEndpoints(key)

	switch eventType {
	case watch.Added, watch.Modified:
		glog.V(4).Infof("Modifying endpoints for %s", key)
		routerEndpoints := createRouterEndpoints(endpoints)
		key := endpointsKey(*endpoints)
		p.Router.AddEndpoints(key, routerEndpoints)
	}

	return p.Router.Commit()
}

// HandleRoute processes watch events on the Route resource.
func (p *TemplatePlugin) HandleRoute(eventType watch.EventType, route *routeapi.Route) error {
	key := routeKey(*route)
	if _, ok := p.Router.FindServiceUnit(key); !ok {
		glog.V(4).Infof("Creating new frontend for key: %v", key)
		p.Router.CreateServiceUnit(key)
	}

	switch eventType {
	case watch.Added, watch.Modified:
		glog.V(4).Infof("Modifying routes for %s", key)
		p.Router.AddRoute(key, route)
	case watch.Deleted:
		glog.V(4).Infof("Deleting routes for %s", key)
		p.Router.RemoveRoute(key, route)
	}

	return p.Router.Commit()
}

// TODO: the internal keys for routes and endpoints should be namespaced.  Currently
// there is an upstream issue where the namespace is not set on non-resolved endpoints.
// A fix has been submitted and we should consume it in the next rebase.

// routeKey returns the internal router key to use for the given Route.
func routeKey(route routeapi.Route) string {
	return route.ServiceName
}

// endpointsKey returns the internal router key to use for the given Endpoints.
func endpointsKey(endpoints kapi.Endpoints) string {
	return endpoints.Name
}

// endpointFromString parses the string into host/port and create an endpoint from it.
// if the string is empty then nil, false will be returned
func endpointFromString(s string) (ep *Endpoint, ok bool) {
	if len(s) == 0 {
		return nil, false
	}

	ep = &Endpoint{}
	//not using net.url here because it doesn't split the port out when parsing
	if strings.Contains(s, ":") {
		eArr := strings.Split(s, ":")
		ep.IP = eArr[0]
		ep.Port = eArr[1]
	} else {
		ep.IP = s
		ep.Port = "80"
	}

	ep.ID = ep.IP + ":" + ep.Port

	return ep, true
}

// createRouterEndpoints creates openshift router endpoints based on k8s endpoints
func createRouterEndpoints(endpoints *kapi.Endpoints) []Endpoint {
	routerEndpoints := make([]Endpoint, len(endpoints.Endpoints))

	for i, e := range endpoints.Endpoints {
		ep, ok := endpointFromString(e)

		if !ok {
			glog.Warningf("Unable to convert %s to endpoint", e)
			continue
		}
		routerEndpoints[i] = *ep
	}

	return routerEndpoints
}
