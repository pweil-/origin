package templaterouter

import (
	"testing"
	routeapi "github.com/openshift/origin/pkg/route/api"
)

func emptyRouter() templateRouter {
	return templateRouter{state: map[string]ServiceUnit{}}
}


func TestCreateServiceUnit(t *testing.T){
	router := emptyRouter()
	suKey := "test"
	router.CreateServiceUnit("test")

	if _, ok := router.FindServiceUnit(suKey); !ok {
		t.Errorf("Unable to find serivce unit %s after creation", suKey)
	}
}

func TestDeleteServiceUnit(t *testing.T){
	router := emptyRouter()
	suKey := "test"
	router.CreateServiceUnit(suKey)

	if _, ok := router.FindServiceUnit(suKey); !ok {
		t.Errorf("Unable to find serivce unit %s after creation", suKey)
	}

	router.DeleteServiceUnit(suKey)

	if _, ok := router.FindServiceUnit(suKey); ok {
		t.Errorf("Service unit %s was found in state after delete", suKey)
	}
}

func TestAddEndpoints(t *testing.T){
	router := emptyRouter()
	suKey := "test"
	router.CreateServiceUnit(suKey)

	if _, ok := router.FindServiceUnit(suKey); !ok {
		t.Errorf("Unable to find serivce unit %s after creation", suKey)
	}

	endpoint := Endpoint {
		ID: "ep1",
		IP: "ip",
		Port: "port",
	}

	router.AddEndpoints(suKey, []Endpoint{endpoint})

	su, ok := router.FindServiceUnit(suKey)

	if !ok {
		t.Errorf("Unable to find created service unit %s", suKey)
	} else {
		if len(su.EndpointTable) != 1 {
			t.Errorf("Expected endpoint table to contain 1 entry")
		} else {
			actualEp, ok := su.EndpointTable[endpoint.ID]

			if !ok {
				t.Errorf("Endpoint %s was not found", endpoint.ID)
			} else {
				if endpoint.IP != actualEp.IP || endpoint.Port != actualEp.Port {
					t.Errorf("Expected endpoint %v did not match actual endpoint %v", endpoint, actualEp)
				}
			}
		}
	}
}

func TestDeleteEndpoints(t *testing.T){
	router := emptyRouter()
	suKey := "test"
	router.CreateServiceUnit(suKey)

	if _, ok := router.FindServiceUnit(suKey); !ok {
		t.Errorf("Unable to find serivce unit %s after creation", suKey)
	}

	router.AddEndpoints(suKey, []Endpoint{
			{
				ID: "ep1",
				IP: "ip",
				Port: "port",
			},
		})

	su, ok := router.FindServiceUnit(suKey)

	if !ok {
		t.Errorf("Unable to find created service unit %s", suKey)
	} else {
		if len(su.EndpointTable) != 1 {
			t.Errorf("Expected endpoint table to contain 1 entry")
		} else {
			router.DeleteEndpoints(suKey)

			su, ok := router.FindServiceUnit(suKey)

			if !ok {
				t.Errorf("Unable to find created service unit %s", suKey)
			} else {
				if len(su.EndpointTable) > 0 {
					t.Errorf("Expected endpoint table to be empty")
				}
			}
		}
	}
}

func TestRouteKey(t *testing.T){
	router := emptyRouter()
	route := &routeapi.Route{
		Host: "test",
	}

	key := router.routeKey(route)

	if key != "test-" {
		t.Errorf("Expected key 'test' but got: %s", key)
	}

	route.Path = "test2"

	key = router.routeKey(route)

	if key != "test-test2" {
		t.Errorf("Expected key 'test' but got: %s", key)
	}
}

func TestAddRoute(t *testing.T){
	router := emptyRouter()
	route := &routeapi.Route{
		Host: "host",
		Path: "path",
		TLS: routeapi.TLSConfig {
			Termination: routeapi.TLSTerminationEdge,
			Certificate: []byte("abc"),
			Key: []byte("def"),
			CACertificate: []byte("ghi"),
			DestinationCACertificate: []byte("jkl"),
		},
	}
	suKey := "test"
	router.CreateServiceUnit(suKey)

	router.AddRoute(suKey, route)

	su, ok := router.FindServiceUnit(suKey)

	if !ok {
		t.Errorf("Unable to find created service unit %s", suKey)
	} else {
		routeKey := router.routeKey(route)
		saCfg, ok := su.ServiceAliasConfigs[routeKey]

		if !ok {
			t.Errorf("Unable to find created serivce alias config for route %s",routeKey )
		} else {
			if saCfg.Host != route.Host || saCfg.Path != route.Path || !compareTLS(route, saCfg, t){
				t.Errorf("Route %v did not match serivce alias config %v", route, saCfg)
			}
		}
	}
}

func compareTLS(route *routeapi.Route, saCfg ServiceAliasConfig, t *testing.T) bool {
	return findCert(route.TLS.DestinationCACertificate, saCfg.Certificates, false, t) &&
			findCert(route.TLS.CACertificate, saCfg.Certificates, false, t) &&
			findCert(route.TLS.Key, saCfg.Certificates, true, t) &&
			findCert(route.TLS.Certificate, saCfg.Certificates, false, t)
}

func findCert(cert []byte, certs map[string]Certificate, isPrivateKey bool, t *testing.T) bool {
	found := false
	sCert := string(cert)

	for _, c := range certs{
		if isPrivateKey {
			if string(c.PrivateKey) == sCert {
				found = true
				break
			}
		} else {
			if string(c.Contents) == sCert {
				found = true
				break
			}
		}
	}

	if !found {
		t.Errorf("unable to find cert %s in %v", sCert, certs)
	}

	return found
}

func TestRemoveRoute(t *testing.T){
	router := emptyRouter()
	route := &routeapi.Route{
		Host: "host",
	}
	suKey := "test"
	router.CreateServiceUnit(suKey)

	router.AddRoute(suKey, route)

	su, ok := router.FindServiceUnit(suKey)

	if !ok {
		t.Errorf("Unable to find created service unit %s", suKey)
	} else {
		routeKey := router.routeKey(route)
		saCfg, ok := su.ServiceAliasConfigs[routeKey]

		if !ok {
			t.Errorf("Unable to find created serivce alias config for route %s",routeKey )
		} else {
			if saCfg.Host != route.Host || saCfg.Path != route.Path {
				t.Errorf("Route %v did not match serivce alias config %v", route, saCfg)
			} else {
				router.RemoveRoute(suKey, route)

				su, _ := router.FindServiceUnit(suKey)
				_, ok := su.ServiceAliasConfigs[routeKey]

				if ok {
					t.Errorf("Route %v was expected to be deleted but was still found", route)
				}
			}
		}
	}
}
