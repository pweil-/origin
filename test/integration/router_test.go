// +build integration,!no-docker,docker
package integration

import (
	"os"
	"net/http"
	"crypto/tls"
	"errors"
	"testing"

	dockerClient "github.com/fsouza/go-dockerclient"
	tr "github.com/openshift/origin/test/integration/router"
	routeapi "github.com/openshift/origin/pkg/route/api"
	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
	"encoding/json"
	"time"
)

const defaultRouterImage = "openshift/origin-haproxy-router"

func init() {
	requireDocker()
}

func TestRouter(t *testing.T){
	//create a server which will act as a user deployed application that
	//serves http and https as well as act as a master to simulate watches
	fakeMasterAndPod := tr.NewTestHttpServer()
	defer fakeMasterAndPod.Stop()

	err := fakeMasterAndPod.Start();
	validateServer(fakeMasterAndPod, t)

	if err != nil {
		t.Fatalf("Unable to start http server: %v", err)
	}

	//deploy router docker container
	dockerCli, err := newDockerClient()

	if err != nil {
		t.Fatalf("Unable to get docker client: %v", err)
	}

	routerId, err := createAndStartRouterContainer(dockerCli, fakeMasterAndPod.MasterHttpAddr)

	if err != nil {
		t.Fatalf("Error starting container %s : %v", getRouterImage(), err)
	}

	defer cleanUp(dockerCli, routerId)

	//run through test cases now that environment is set up
	testCases := []struct {
		name string
		serviceName string
		endpoints []string
		routeAlias string
		endpointEventType watch.EventType
		routeEventType watch.EventType
		protocol string
	    expectedResponse string
		routeTLS *routeapi.TLSConfig
	}{
		{
			name: "non-secure",
			serviceName: "example",
			endpoints: []string{fakeMasterAndPod.PodHttpAddr},
			routeAlias: "www.example-unsecure.com",
			endpointEventType: watch.Added,
			routeEventType: watch.Added,
			protocol: "http",
			expectedResponse: tr.HelloPod,
			routeTLS: nil,
		},
		{
			name: "edge termination",
			serviceName: "example-edge",
			endpoints: []string{fakeMasterAndPod.PodHttpAddr},
			routeAlias: "www.example.com",
			endpointEventType: watch.Added,
			routeEventType: watch.Added,
			protocol: "https",
			expectedResponse: tr.HelloPod,
			routeTLS: &routeapi.TLSConfig{
				Termination: routeapi.TLSTerminationEdge,
				Certificate: tr.ExampleCert,
				Key: tr.ExampleKey,
				CACertificate: tr.ExampleCACert,
			},
		},
		{
			name: "passthrough termination",
			serviceName: "example-passthrough",
			endpoints: []string{fakeMasterAndPod.PodHttpsAddr},
			routeAlias: "www.example2.com",
			endpointEventType: watch.Added,
			routeEventType: watch.Added,
			protocol: "https",
			expectedResponse: tr.HelloPodSecure,
			routeTLS: &routeapi.TLSConfig{
				Termination: routeapi.TLSTerminationPassthrough,
			},
		},
	}

	routerUrl := "0.0.0.0"

	for _, tc := range testCases {
		//simulate the events
		endpointEvent := &watch.Event{
			Type: tc.endpointEventType,

			Object: &kapi.Endpoints{
				ObjectMeta: kapi.ObjectMeta{
					Name: tc.serviceName,
				},
				TypeMeta: kapi.TypeMeta{
					Kind: "Endpoints",
					APIVersion: "v1beta3",
				},
				Endpoints: tc.endpoints,
			},
		}

		routeEvent := &watch.Event{
			Type: tc.routeEventType,
			Object: &routeapi.Route{
				TypeMeta: kapi.TypeMeta{
					Kind: "Route",
					APIVersion: "v1beta1",
				},
				Host: tc.routeAlias,
				ServiceName: tc.serviceName,
				TLS: tc.routeTLS,
			},
		}

		fakeMasterAndPod.EndpointChannel <- eventString(endpointEvent)
		fakeMasterAndPod.RouteChannel <- eventString(routeEvent)

		//allow the router time to pick up and process the watches
		time.Sleep(time.Second * 5)

		//now verify the route with an http client
		resp, err := getRoute(routerUrl, tc.routeAlias, tc.protocol)

		if err != nil {
			t.Errorf("Unable to verify response: %v", err)
		}

		var respBody = make([]byte, len([]byte(tc.expectedResponse)))
		resp.Body.Read(respBody)

		if string(respBody) != tc.expectedResponse {
			t.Errorf("TC %s failed! Response body %v did not match expected %v", tc.name, string(respBody), tc.expectedResponse)
		}
	}
}

func getRoute(routerUrl string, hostName string, protocol string) (response *http.Response, err error){
	url := protocol + "://" + routerUrl
	var httpClient *http.Client

	if protocol == "https" {
		secureTransport := &http.Transport {
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				ServerName: hostName,
			},
		}
		httpClient = &http.Client{Transport: secureTransport}

	} else {
		httpClient = &http.Client{}
	}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	req.Host = hostName
	resp, err := httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func eventString(e *watch.Event) string {
	s, _ := json.Marshal(e)
	return string(s)
}

func createAndStartRouterContainer(dockerCli *dockerClient.Client, masterIp string) (containerId string, err error) {
	ports := []string{"80", "443"}
	portBindings := make(map[dockerClient.Port][]dockerClient.PortBinding)
	exposedPorts := map[dockerClient.Port]struct{}{}

	for _, p := range ports {
		dockerPort := dockerClient.Port(p + "/tcp")

		portBindings[dockerPort] = []dockerClient.PortBinding{
			dockerClient.PortBinding{
				HostPort: p,
			},
		}

		exposedPorts[dockerPort] = struct{}{}
	}

	containerOpts := dockerClient.CreateContainerOptions {
		Config: &dockerClient.Config{
			Image: getRouterImage(),
			Cmd: []string{"--master=" + masterIp, "--loglevel=4"},
			ExposedPorts: exposedPorts,
		},
	}

	container, err := dockerCli.CreateContainer(containerOpts)

	if err != nil {
		return "", err
	}

	dockerHostCfg := &dockerClient.HostConfig{NetworkMode: "host", PortBindings: portBindings}
	err = dockerCli.StartContainer(container.ID, dockerHostCfg)

	if err != nil {
		return "", err
	}

	running := false

	//wait for it to start
	for i := 0; i < 3; i++{
		c, err := dockerCli.InspectContainer(container.ID)

		if err != nil {
			return "", err
		}

		if c.State.Running {
			running = true
			break
		}
		time.Sleep(time.Second * 2)
	}

	if !running {
		return "", errors.New("Container did not start after 3 tries!")
	}

	return container.ID, nil
}

func validateServer(server *tr.TestHttpServer, t *testing.T){
	_, err := http.Get("http://" + server.MasterHttpAddr)

	if err != nil {
		t.Errorf("Error validating master addr %s : %v", server.MasterHttpAddr, err)
	}

	_, err = http.Get("http://" + server.PodHttpAddr)

	if err != nil {
		t.Errorf("Error validating master addr %s : %v", server.MasterHttpAddr, err)
	}

	secureTransport := &http.Transport {TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	secureClient := &http.Client{Transport: secureTransport}
	_, err = secureClient.Get("https://" + server.PodHttpsAddr)

	if err != nil {
		t.Errorf("Error validating master addr %s : %v", server.MasterHttpAddr, err)
	}
}

func cleanUp(dockerCli *dockerClient.Client, routerId string){
	dockerCli.StopContainer(routerId, 5)

	dockerCli.RemoveContainer(dockerClient.RemoveContainerOptions{
		ID: routerId,
		Force: true,
	})
}

func getRouterImage() string {
	i := os.Getenv("OPENSHIFT_ROUTER_IMAGES")

	if len(i) == 0 {
		i = defaultRouterImage
	}

	return i
}
