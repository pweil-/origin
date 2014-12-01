- how are shards configured
- how is router configuration visualized from a user perspective
- how is a shard chosen for a route
- how is a user notified of a route allocation and final dns
- how does a user request default dns name vs custom dns name
- re-allocation, does it ever occur
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

## Creating Sharded Routers

Options for creating a sharded router must answer the following questions:
 
- Where should config live?
- How are shards configured?
- How does OpenShift know about routers (for route allocation)?

#### Option 1: Router is administered as a pod

Administering the router as a pod and *NOT* as a custom top level object allows a quick
implementation but reduces the ability to visualize the router infrastructure and deal with it with
commands specific to routers.  This option means that configuration visualization needs to be
provided through the `inspect` or `describe` commands or by providing hooks into the router
containers or storage mechanisms (etcd) so that routers can be visualized as a complete unit or
as individual routers.

- Where does configuration live: etcd, just like any other pod
- How are shards configured: shards are configured via environment variables
- How does OpenShift know about routers (for route allocation): routers must be registered with 
  OpenShift via convention  or configuration

Pros: 

- Default infra 
- Less custom code, config syntax fits nicely into container env vars

Cons: 

- Unable to provide custom commands and visualization through the CLI
- The system doesn't know about routers by default, they need to be registered somehow for route
  allocation

#### Option 2: Router is a top level object

Administering routers as a top level object allows administrators to use custom commands specific
to routers.  This provides a more use friendly mechanism of configuration and customizing routers.
However, this also introduces more code for  an object that will likely be dealt with as a pod
anyway.  Routers should be a low touch configuration item that do not require many custom commands
for daily administration.

- Where does configuration live: etcd, just like any other pod
- How are shards configured: shards are configured via custom commands and `json` syntax
- How does OpenShift know about routers (for route allocation): routers are known to OpenShift at
  create time

Pros: 

- Custom administration syntax
- Deal with routers as infra
- The system knows about routers for route allocation with no extra effort

Cons: 

- More divergent from Kubernetes codebase which is generally bad

#### Option 3: Hybrid

Routers could be administered with custom commands in the OpenShift client and still *NOT* be top
level objects.  This allows administrators to create and administer routers via the CLI, allows the
system to explicitly know about routers when they are created, and allows more robust visualization
commands.

- Where does configuration live: etcd, just like any other pod
- How are shards configured: shards are configured via environment variables as in the pod scenario
  and only created with custom commands like `create router`
- How does OpenShift know about routers (for route allocation): routers are known to OpenShift at
  create time

Pros: 

- Custom administration syntax
- Deal with routers as infra
- The system knows about routers for route allocation with no extra effort

Cons: 

- Possibly confusing that you can administer a router via custom commands or the existing pod
  commands

## User Requests a Route

when they have their own dns name then need to point it to our nameservers, map their dns to a shard name with a c record?
when requesting default dns we should take the name, allocate it, and provide then a final dns name?


## Create DNS

Option 1: internal dns impl that syncs with routes
Option 2: manual 


