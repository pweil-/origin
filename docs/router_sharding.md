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
However, this also introduces more code for an object that will likely be dealt with as a pod
anyway.  Routers should be a low touch configuration item that do not require many custom commands
for daily administration.

Pros:

- Configuration lives in etcd, just like any other resource
- Shards are configured via custom commands and `json` syntax
- Routers are known to OpenShift; the system ensures the proper configuration is running
- Custom administration syntax
- Deal with routers as infra
- The system knows about routers for route allocation and visualization with no extra effort

Cons: 

- More divergent from Kubernetes codebase initially, though we may be able to generalize parts of
  this approach to sharding to other resources and controllers which allow sharding

### Proposed Implementation

#### The `Router` Resource

There should be a new OpenShift resource called `Router`.  Its fields include:

1.  `Name`: the router's name
2.  `Description`: a description of the Router
3.  `Label`: the label that associates resources (Endpoints, Routes, Services) with this router

## Route Allocation

Route allocation is the process of assigning a `Route` record to a specific `Router` and setting up
DNS for routes.  We will treat the problem of route allocation similarly to the problem of
scheduling a Pod.  There will be a new state reconciler to allocate Routes after they are created
and a new field to express allocation status in the `Route` resource.

### Proposed Implementation

#### Changes to the `Route` resource

The `Route` resource should have a new field added:

    type Route {
        // other fields not shown
        RouterURL        string
        AllocationStatus RouteAllocationStatus
    }

The `RouteAllocationStatus` type represents the allocation status of a route; it can be valued
`new` or `allocated`.

#### Changes to the `Route` REST API

The `Route` REST API will be changed to validate that:

1.  The `RouterURL` and `AllocationStatus` fields of a `Route` are not set during create
2.  The value of `RouterURL` and `AllocationStatus` fields do not change during update

#### The `RouteAllocation` Resource

The `RouteAllocation` resource describes the association of a `Route` with a `Router`.  Its fields
are:

1.  `RouteNamespace`: The namespace of the route being allocated
2.  `RouteName`: The name of the route being allocated
3.  `RouterNamespace`: The namespace of the router serving the route
4.  `RouterName`: The name of the router serving the route

The `RouteAllocation` REST API will be the only path that is allowed to update the values of the
`RouterURL` and `AllocationStatus` fields.  The REST API will apply the allocation to the `Route`
record during `Create`.

#### The `RouteAllocator` state reconciler

We will introduce `RouteAllocator`, a state reconciler that watches the `Route` resource and
allocates new routes.  The allocator will use a pluggable allocation strategy, allowing users to
author their own strategies.  Our initial strategy implementation will be a simple round-robin
strategy.

The `RouteAllocator` processes `Route` resources as follows:

1.  The `RouteAllocator` will watch for newly created (and thus unallocated) `Route`s and
    periodically list the unallocated `Route`s to retry
2.  The allocator passes unallocated `Route` records to the `RouteAllocationStrategy` interface
3.  The allocator creates a `RouteAllocation` for the  route and router if the allocation is
    successful
4.  The `RouteAllocation` REST API applies the allocation to the `Route`, setting the `RouterURL`
    field
5.  The `Router` instance the `Route` is allocated to will receive an update event for the route
    and apply it to the router backend configuration

Errors allocating routes are assumed to be transient and actionable by administrators.  The
allocator will continue reprocessing a `Route` until allocation succeeds.

#### The `RouteAllocationStrategy` interface

The `RouteAllocationStrategy` expresses something that can allocate routes amongst the available
routers:

    type RouteAllocationStrategy interface {
        func AllocateRoute(*routeapi.Route) (*routerapi.Router, error)
    }

## User Requests a Route

when they have their own dns name then need to point it to our nameservers, map their dns to a shard name with a c record?
when requesting default dns we should take the name, allocate it, and provide then a final dns name?

## DNS

OPEN QUESTION: do we intend on hosting DNS for the Online use case?

1. NO: users map their domain to resolve our router ip(s).  Must be done after allocation.  User is responsible for balancing requests between routers?
2. YES: users configure their domain to point to our nameservers for resolution.  Can be done before allocation (nameserver IPs are known).  Allows us to add
routers to shards and have them picked up by DNS RR.  For custom DNS we make a CNAME that points to the wildcard shard entry
3. BOTH: we are still dealing with DNS cache issues

In order to facilitate supplying external DNS for applications in the OpenShift system the router configuration will be 
modified with an indicator that the DNS name is user owned or system controlled.

     {
        "type": "route",
        ...
        "dnsType": "system|user",
     }

1.  System supplied DNS: this indicates that the user *DOES NOT* own the domain name and is requesting that OpenShift 
supply it.  The user provides a `Host` that is used as a prefix to the final DNS name which is determined based on the router 
allocation and takes the form of: `<namespace>-<Host>.<shard>.v3.rhcloud.com.

1.  User supplied DNS: this indicates that the user currently owns a domain name and will be able to configure their 
registrar to indicate that OpenShift's DNS servers will provide DNS look ups for the domain.  When a user controlled DNS 
entry is request no manipulation will be done to the `Host` field of the `route` configuration.


#### DNS Implementations

DNS plugins will be able to watch the `router` configuration to determine the correct zone files to set up with wildcard 
entries.  It will also be able to watch the `route` configuration to make entries for user supplied DNS requests that map 
to a shard.

Example: 
    
    shard1.zone:
    $ORIGIN shard1.v3.rhcloud.com.
    
    @       IN      SOA     . shard1.v3.rhcloud.com. (
                         2009092001         ; Serial
                             604800         ; Refresh
                              86400         ; Retry
                            1206900         ; Expire
                                300 )       ; Negative Cache TTL
            IN      NS      ns1.v3.rhcloud.com.
    ns1     IN      A       127.0.0.1
    *       IN      A       10.245.2.2      ; active/active DNS round robin
            IN      A       10.245.2.3      ; active/active DNS round robin
            
    shard2.zone:
    $ORIGIN shard2.v3.rhcloud.com.
    
    @       IN      SOA     . shard2.v3.rhcloud.com. (
                         2009092001         ; Serial
                             604800         ; Refresh
                              86400         ; Retry
                            1206900         ; Expire
                                300 )       ; Negative Cache TTL
            IN      NS      ns1.v3.rhcloud.com.
    ns1     IN      A       127.0.0.1
    *       IN      A       10.245.2.4      ; active/active DNS round robin
            IN      A       10.245.2.5      ; active/active DNS round robin 
                       
    user_supplied.zone:
    $ORIGIN example.com.
    
    @       IN      SOA     . example.com. (
                         2009092001         ; Serial
                             604800         ; Refresh
                              86400         ; Retry
                            1206900         ; Expire
                                300 )       ; Negative Cache TTL
            IN      NS      ns1.v3.rhcloud.com.
    ns1     IN      A       127.0.0.1
    www     IN      CNAME   shard1.v3.rhcloud.com ; points to shard                           





