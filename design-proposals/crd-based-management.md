# CRD-based management

**Authors**: @elevran, @oro

**Begin Design Discussion**: 2023-11-22

**(optional) Status:** draft

## Summary/Abstract

This design proposal explores the design space for managing ClusterLink through
 Kubernetes (k8s) Custom Resource Definitions (CRDs). It aims to replace the
 current API and CLI with k8s native management philosophy (e.g., asynchronous
 reconciliation of actual and desired states) and tooling (e.g., `kubectl`, RBAC).
 This document focuses on single cluster management. A central management solution
 could be considered in a future extension. In the context of the current design,
 information needed by a cluster regarding remote peers (e.g., their gateways,
 services they expose, etc.) ise assumed to materialize by the user out of thin
 air/magic (TODO reword... Or it may make sense to have a short paragraph upfront
 describing what an imaginary CM may provide?).

## Background

### Motivation and problem space

Currently ClusterLink exposes a RESTful API for its users. A convenience CLI (`gwctl`)
 is provided for easier API access. The API/CLI is based on Created/Read/Update/Delete
 (CRUD) of the following objects:

- Remote `Peer` clusters this cluster can connect to
- Service `Export`s this cluster makes available for consumption to remote clusters
- Service `Import`s that this cluster makes avaialble to local workloads (TODO: `Binding`
 objects are needed? Are they fundamental to the mental model?)
- `Policy` objects governing aspects of the cross cluster communication.

An additional CLI (`cl-adm`) is provided to ease the bootstrap and deployment of ClusterLink
 in a k8s cluster. The `cl-adm` utility creates a set of (mostly k8s configuration) files on
 the local file systems which can be used to ...

TODO Lack of users and permissions, need to support tooling with relatively low value.
  cl-adm and gwctl are not consistent.

### Impact and desired outcome

TODO The reverse of some of the above...

### Prior discussion and links

TODO Some initial ideas in GH issue. Reference.

## (Optional) User/User Story

### Users

Multiple users

- Fabric admin: Control over entire fabric (e.g, create fabric, add peers to fabric);
 mostly not in scope for this design doc
- Site admin: Control a single cluster. Deploys CL pods to the cluster to a specific namespace. Add existing peers. Add global policies (e.g. allow only a certain namespace to access a specific peer). Has full control of the kubernetes cluster (all namespaces).
- Local importer/exporter (aka user): Creates import, exports, and policies for them. In most cases, RBAC will limit this user to work in a single unique namespace. We will support users from multiple namespaces.

### Export local service user story

User wants to export a single-port local service "foo" in namespace X.

To do so, it creates an Export k8s object whose name matches the name of the exported service (i.e. "foo").

The status of the export can reflect the following errors:
- The target service does not exists.
- The target service has more/less than a single port.

A pre-condition for the export to be consumed is that the actual service port is reported as part of the export status. 

If any changes are made to the target service, the export status is expected to update accordingly.

### Export multi-port local service user story

User wants to export one or more ports from a multi-port local service "foo" in namespace X.

To do so, it needs to create an Export k8s object for each port to be exported.
The export object spec will include the name of the service ("foo") and the desired exported port.
In case the service name is not specified in the spec, it is assumed to match the export name.

The status of the export can reflect the following errors:
- The target service does not exists.
- The requested port is not a valid port in the requested service.

Like in the single-port case above, the port (copied from the spec) is reported as part of the export status.
If any changes are made to the target service, the export status is expected to update accordingly.

### Export endpoint user story

User wants to export some service which is accessible from within a cluster via an endpoint host/ip:port.
In particular, this can be used to export a service from any namespace (e.g. my-service.my-namespace.svc.cluster.local).

To do so, the user needs to create an Export k8s object whose spec includes the service host/ip and port.

In this case, the export status is empty, as no validation is made against the endpoint.

**QUESTION**: do we want to try to DNS lookup the service host, and post a failure in the status?

### Peer creation user story

Site A admin wants to allow users to import services from site B.

To do so, it creates a Peer k8s object for site B.
The name of the object will be used by importers as part of the routable name of an export.
The peer spec includes one or more gateways (host/ip,port) for accessing the peer's dataplane server.

**QUESTION**: Should we add an expected server name to the spec, or rely just on the gateway hosts?

The status of the peer object will report whether the peer is currently reachable,
and the last time it was successfully reached (via heartbeats).

### Import service user story

User of site B wants to import an exported service "foo" from namespace X of site A,
into namespace Y of site B.

To do so, it creates an Import k8s object in the desired namespace Y.
The spec of the import includes one or more exports which serve as the source of the import.
Each export is identified using its routable name: (peer name, export namespace, export name).

By default, any access to the imported service will be randomly forwarded to one of the exports.
The spec will also include customization of the load-balancing policy that is used
to select the target export per each incoming connection to the imported service.

The imported service will be accessible via a new k8s service matching the (namespace, name)
of the import object.
Any existing service matching that name will be overriden.

The status of the import can reflect the following errors:
- One or more of the exports mentioned do no exist / unreachable.
- The k8s service for the import cannot be created.

### Export/import policy user story

See relevant policy design document.

## Goals

<!-- List the desired goal or goals that the design is intended to achieve. These goals can be
 leveraged to structure and scope the design and discussions and may be reused as the
 "definition of done" -  the criteria you use to know the implementation of the design
 has succeeded in accomplishing its goals. -->

Full life cycle management of clusterlink configuration, compatible with CLI functionality.
All available via kubectl.
Automation of common tasks (at least potential for it)

## Non-Goals

<!-- Describe what is out of scope for the design proposal. Listing non-goals helps to focus
 discussion and make progress. -->

Central management (TODO consider discussing CM here?)
Multitenancy support (multiple fabrics)
Per-namespace/unprivileged

All of the above can be future extensions, so make sure to not block them?

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is. It should
 have enough detail that reviewers can understand exactly what you're proposing, but
 should not include things like API designs or implementation. This section should expand
 on the desired outcome and include details on how to measure success. -->

## Design Details

- Use of namespaces (`clusterlink-system` separate from where service run. Discuss implications
 of putting configration )
- Definition of users/roles/permissions
- Overall API concepts (k8s Spec, Status. Status to contain Conditions, use of references,
 all objects are namespace-scoped, etc. async completion of configuration changes (status
 and conditions, etc.))
- Section for each CRD type, with Go structure definition, discussion of alternatives
  - Fabric: name, CA keypair
  - Peer: keypair, gateways
  - Remote peers: collection of Peers (without keypair)
  - Export: internal service name (or object reference if out of namespace) and port,
   external "monikor", ... Is it needed or are ACL policies enough? Exports is
   explicit, ...
  - Import: ...
  - Binding: is it needed?
  - Policy ...

<!-- 
This section should contain enough information to allow the following to occur:

- potential contributors understand how the feature or change should be implemented
- users or operators understand how the feature of change is expected to function and
 interact with other components of the project
- users or operators can take action to pre-plan any needed changes within their
 architecture that impacted by the upcoming feature or change if it's approved for
 implementation
- decisions or opinions on a specific approach are fully discussed and explained
- users, operators, and contributors can gain a comprehensive understanding of
 compatibility of the feature or change with past releases of the project.

This may include API specs (though not always required), code snippets, data flow
 diagrams, sequence diagrams, etc.

If there's any ambiguity about HOW your proposal will be implemented, this is the place
 to discuss them. This can also be combined with the proposal section above. It should
 also address how the solution is backward compatible and how to deal with these
 incompatibilities, possibly with defaulting or migrations. It may be useful to refer
 back to the goals and non-goals to assist in articulating the "why" behind your approach.
-->

## Impacts / Key Questions

<!-- List crucial impacts and key questions, some of which may still be open. They likely
 require discussion and are required to understand the trade-offs of the design. During
 the lifecycle of a design proposal, discussion on design aspects can be moved into this
 section. After reading through this section, it should be possible to understand any
 potentially negative or controversial impact of the design. It should also be possible
 to derive the key design questions: X vs Y.

This will also help people understand the caveats to the proposal, other important
 details that didn't come across above, and alternatives that could be considered. It can
 also be a good place to talk about core concepts and how they relate. It can be helpful
 to explicitly list the pros and cons of each decision. Later, this information can be
 reused to update project documentation, guides, and Frequently Asked Questions (FAQs).
-->

- List our major assumptions and discuss them? single tenant, privileged/cluster-scope
- net-admin, user role or combination
- cluster, namespace, multiple-namespaces support

<!-- TODO: do we need these sections?
### Pros

Pros are defined as the benefits and positive aspects of the design as described. It
 should further reinforce how and why the design meets its goals and intended outcomes.
 This is a good place to check for any assumptions that have been made in the design.

### Cons

Cons are defined as the negative aspects or disadvantages of the design as described.
 This section has the potential to capture outstanding challenge areas or future
 improvements needed for the project and could be referenced in future PRs and issues.
 This is also a good place to check for any assumptions that have been made in the design.
-->

## Risks and Mitigations

<!--
Describe the risks of this proposal and how they can be mitigated. This should be broadly
 scoped and describe how it will impact the larger ecosystem and potentially adopters of
 the project; such as if adopters need to immediately update, or support a new port or
 protocol. It should include drawbacks to the proposed solution.-->

- Non k8s focus or users (run k3s or similar to mitigate - at least in VM/BM setting)
- others?

### Security Considerations

<!--
When attempting to identify security implications of the changes, consider the following questions:

- Does the change alter the permissions or access of users, services, components - this
 could be an improvement or downgrade or even just a different way of doing it?
- Does the change alter the flow of information, events, and logs stored, processed, or
 transmitted?
- Does the change increase the 'surface area' exposed - meaning, if an operator of the
 project or user were to go rogue or be uninformed in its operation, do they have more
 areas that could be manipulated unfavorably?
- What existing security features, controls, or boundaries would be affected by this
 change?

This section can also be combined into the one above.
-->

I think this design is mostly an improvement (security wise) over the current model.
 Are there glaring holes in it? Beyond k8s in general (users compromising the cluster's API
 server, etc.). For example, the clusterlink user might have broad permission? Users may
 want to explicitly disallow certain namepsace (e.g., k8s-system) - should this be via
 RBAC or CL configuration?
 