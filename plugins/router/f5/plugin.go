package f5

import (
	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
	"github.com/golang/glog"

	routeapi "github.com/openshift/origin/pkg/route/api"
)

type F5Plugin struct {
	//this should be the restful client for the f5 router, string is just
	//a placeholder.  The two Handle methods below are intended to delegate to
	//the client which should hold state and do whatever necessary to configure
	//the F5 router
	F5Client string
}

// NewF5Plugin creates a new F5Plugin.
func NewF5Plugin(f5Client string) (*F5Plugin, error) {
	return &F5Plugin{f5Client}, nil
}

// HandleEndpoints processes watch events on the Endpoints resource.
func (p *F5Plugin) HandleEndpoints(eventType watch.EventType, endpoints *kapi.Endpoints) error {
	glog.V(4).Infof("Processing %d Endpoints for Name: %v (%v)", len(endpoints.Endpoints), endpoints.Name, eventType)

	for i, e := range endpoints.Endpoints {
		glog.V(4).Infof("  Endpoint %d : %s", i, e)
	}

	//do whatever f5 needs to do, see the template plugin for examples
	//if we want to use the same keying mechanisms as the template router we can refactor the
	//utility methods out to a separate class
	return nil
}

// HandleRoute processes watch events on the Route resource.
func (p *F5Plugin) HandleRoute(eventType watch.EventType, route *routeapi.Route) error {
	glog.V(4).Infof("Processing route for service: %v (%v)", route.ServiceName, route)
	//do whatever f5 needs to do, see the template plugin for examples
	return nil
}

