- how are shards configured
- how is router configuration visualized from a user perspective
- how is a shard chosen for a route
- how is a user notified of a route allocation and final dns
- how does a user request default dns name vs custom dns name
- re-allocation, does it ever occur
- router fronting with DNS, how are entries created

### Description

As an application administrator, I would like my routes to be configured with shards so they can grow beyond a single
active/active or active/passive setup.  I should be able to configure many routers to allocate user requested
routes to and be able to visualize the configuration.  

### Constraints

1. Router re-allocation is out of scope for this proposal.  When using DNS to front routers you must deal
with the client caching feature that is unpredictable. 


### Use Cases

The following use cases should be satisfied by this proposal:

1. Create multiple routers with shards
1. User requests default route for application
1. User requests custom route for application
1. Create DNS (or other front end entry points) for routers


### Existing Artifacts

Routing: https://github.com/pweil-/origin/blob/master/docs/routing.md
HA Routing: https://github.com/pweil-/origin/blob/master/docs/routing.md#running-ha-routers
DNS Round Robin: https://github.com/pweil-/origin/blob/master/docs/routing.md#dns-round-robin

### Creating Sharded Routers

**Option 1: Router is administered as a pod**

Pros: 

- default infra, 
- less custom code, config syntax fits nicely into container env vars

Cons: 

- unable to provide custom commands and visualization

**Option 2: Router is a top level object**

Pros: 

- custom configuration syntax
- deal with routers as infra

Cons: 

- more divergent from k8s codebase which is generally bad

where should config live?
how are shards configured?
can you re-configure shards once configured?
top level object or pod definition?

### User Requests a Route

when they have their own dns name then need to point it to our nameservers, map their dns to a shard name with a c record?
when requesting default dns we should take the name, allocate it, and provide then a final dns name?


### Create DNS

Option 1: internal dns impl that syncs with routes
Option 2: manual 


