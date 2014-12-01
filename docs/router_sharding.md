- how is router configuration visualized from a user perspective
- how is router configuration visualized from an admin perspective
- how is a shard chosen for a route
- how is a user notified of a route allocation and final dns
- how does a user request default dns name vs custom dns name
- router fronting with DNS, how are entries created

## Description

As an application administrator, I would like my routes to be configured with shards so they can
grow beyond a single active/active or active/passive setup.  I should be able to configure many
routers to allocate user requested routes to and be able to visualize the configuration.  

## Use Cases

The following use cases should be satisfied by this proposal:

1.  Configure routers as OpenShift resources and let the platform keep the specified configuration
    running
1.  Create a single, unsharded router
1.  Create multiple routers with shards corresponding to a resource label
1.  Allow any router to run in an HA configuration
1.  User requests default route for application
1.  User requests custom route for application
1.  Create DNS (or other front end entry points) for routers

## Existing Artifacts

1.  Routing: https://github.com/pweil-/origin/blob/master/docs/routing.md
1.  HA Routing: https://github.com/pweil-/origin/blob/master/docs/routing.md#running-ha-routers
1.  DNS Round Robin: https://github.com/pweil-/origin/blob/master/docs/routing.md#dns-round-robin

## Configuring Routers

Administering routers as a top level object allows administrators to use custom commands specific
to routers.  This provides a more use friendly mechanism of configuration and customizing routers.
However, this also introduces more code for  an object that will likely be dealt with as a pod
anyway.  Routers should be a low touch configuration item that do not require many custom commands
for daily administration.

- Configuration lives in etcd, just like any other resource
- Shards are configured via custom commands and `json` syntax
- Routers are known to OpenShift; the system ensures the proper configuration is running

Pros: 

- Custom administration syntax
- Deal with routers as infra
- The system knows about routers for route allocation and visualization with no extra effort

Cons: 

- More divergent from Kubernetes codebase initially, though we may be able to generalize parts of
  this approach to sharding to other resources and controllers which allow sharding

### Proposed Implementation

#### The `Router` Resource

There should be a new OpenShift resource called `Router`.  Its fields should include:

1.  Type: the type of the router backend to use (HAProxy, nginx	, etc)
2.  Label: the label that associates resources (Endpoints, Routes, Services) with this router
3.  HA: whether this router should be run in HA mode

#### The Router Subsystem State Reconciler

The OpenShift system needs a new state reconciler to ensure that the configured routers are
always running.  This will be a new controller called `RouterSubsystemController` that will watch
the `Router` resource and handle running pods to realize the specified configuration.

## Route Allocation

Route allocation is the process of assigning a `Route` record to a specific `Router` and setting up
DNS for routes.  We will treat the problem of route allocation similarly to the problem of
scheduling a Pod.  There will be a new state reconciler to allocate Routes after they are created
and a new field to express allocation status in the `Route` resource.

### Proposed Implementation

#### `Route` resource changes

The `Route` resource should have a new field added:

    type Route {
    	// other fields not shown
    	Status RouteAllocationStatus
    }

The `RouteAllocationStatus` type represents the allocation status of a route; it can be valued
`NEW` or `ALLOCATED`.

#### Route Allocator

We will introduce `RouteAllocator`, a state reconciler that watches the `Route` resource and allocates new routes.
The allocator will use a pluggable allocation strategy, allowing users to author their own strategies.
Our initial strategy implementation will be a simple round-robin strategy.

## User Requests a Route

when they have their own dns name then need to point it to our nameservers, map their dns to a shard name with a c record?
when requesting default dns we should take the name, allocate it, and provide then a final dns name?


## Create DNS

Option 1: internal dns impl that syncs with routes
Option 2: manual 

