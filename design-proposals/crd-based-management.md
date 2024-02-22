# Custom Resource Based Management of ClusterLink

**Authors**: @elevran, @oro

**Begin Design Discussion**: 2023-11-22

**(optional) Status:** approved

## Summary/Abstract

This design proposal explores the design space for managing ClusterLink through
 Kubernetes (k8s) Custom Resource Definitions (CRDs). It aims to replace the
 current API and CLI with k8s native management philosophy (e.g., asynchronous
 reconciliation of actual and desired states) and tooling (e.g., `kubectl`, RBAC).
 This document focuses on the management of a single cluster and ignores aspects
 relating to centralized or distributed management of a ClusterLink system across
 multiple clusters. In the context of the current design, any information
 needed by a cluster regarding remote peers, such as their gateway identities and
 endpoints, services they expose, etc., is assumed to be exchanged out of band.

## Background

### Motivation and problem space

Currently ClusterLink exposes a RESTful API for its users. A convenience CLI (`gwctl`)
 is provided for easier API access. The API/CLI is based on Created/Read/Update/Delete
 (CRUD) of the following objects:

- The `local ClusterLink peer` (albeit this not made explicit today);
- Remote `Peer` clusters the local peer can interact with;
- Service `Export`s the local peer makes available for consumption to remote clusters;
- Service `Import`s made available to workloads by the local peer (note: the current API
  also includes `Binding` objects, associating an `Import` with `Peer` clusters. This is
  no longer deemed required).
- `Policy` objects governing aspects of the cross cluster communication.

An additional CLI (`cl-adm`) is provided to ease the bootstrap and deployment of ClusterLink
 in a k8s cluster. The `cl-adm` utility creates a set of (mostly k8s configuration) files on
 the local file systems which can be used to bootstrap the ClusterLink installation. This
 mechanism is currently being replaced by an installation operator which is also based on
 CRDs.

Current CLI options provide low level access to API operations (i.e., CRUD on object) and
 consequently relatively low value. In addition, the current API (both `gwctl` and `cl-adm`)
 do not address roles and permissions: `gwctl` users are either authenticated and authorized
 to the entire API or not at all; `cl-adm` files are created locally, relaying on file system
 attributes for protection. The two CLIs are also only partially consistent in UX. Transitioning
 to CRDs and `kubectl` should address both consistency and security using existing k8s support.

## Goals

- Provide full life cycle management of ClusterLink configuration, that is comparable with
 current CLI functionality, but accessible via `kubectl`.
- Enable use of common k8s tooling and management capabilities.

## Non-Goals

- Provide means for central management (e.g., to facilitate fabric management,
 automation of common tasks such as Peer and Service sharing across clusters).
- Support for multitenancy and fabric sharing (i.e., interconnect multiple fabrics)
- Enable per-namespace/unprivileged deployments

All of the above can be future extensions, so make sure to not block them?

### Prior discussion and links

These topics and their related aspects have been discussed, at a high level, in the following issues:

- [25 - CLI harmonization](https://github.com/clusterlink-net/clusterlink/issues/25)
- [28 - K8s Central Management](https://github.com/clusterlink-net/clusterlink/issues/28)
- [38 - gwctl output](https://github.com/clusterlink-net/clusterlink/issues/38)
- [67 - CLI parameters](https://github.com/clusterlink-net/clusterlink/issues/67)
- [68 - Watch Services and Pods](https://github.com/clusterlink-net/clusterlink/issues/68)
- [69 - Control plane persistence](https://github.com/clusterlink-net/clusterlink/issues/69)
- [77 - Privileged deployments](https://github.com/clusterlink-net/clusterlink/issues/77)
- [79 - Async control plane](https://github.com/clusterlink-net/clusterlink/issues/79)
- [133 - Use of k8s controller-runtime](https://github.com/clusterlink-net/clusterlink/issues/133)

## User Stories

### Users

We envisage multiple user roles involved in the management of a ClusterLink deployment.

- **Fabric administrator** has control over the entire ClusterLink fabric. They are responsible
 for creating the fabric's root of trust, controlling which peers are part of the fabric, etc.
 This role is mostly not in scope for this design document.
- **Site administrator** has control over a single cluster in the fabric and manages the
 installation and operation of the local ClusterLink deployment. Site administrators can
 add remote Peers to their cluster and may define admin level network policies.
- **Application owners and developers** manage Services and their access policies. They
 typically have only standard access to a single namespace (i.e., namespace scoped).

TODO: are load balancing decisions done at the Site or Application level?

### User Story: Regulatory Compliance

As a site administrator I would like to ensure only compliant (e.g., EU based for GDPR)
 sites can access services in the cluster I manage. Thus, I would like to (1) specify
  subst of the remote sites that can interconnect with the local cluster; and (2) set
  policies governing access to a sensitive Services, such that access shall be granted
  to workloads only from a specific remote cluster and namespace.

### User Story: Enable Cluster Interconnect

As a site administrator I would like to enable users in my cluster to import Services from
 other clusters the infrastructure team manages. Given the dynamic nature of applications
 run in clusters, I want application owners to manage their own sharing without further
 involvement.

### User Story: Service Sharing

As an application owner, I would like to enable access to a Service I own from namespaces
 assigned to me in other clusters. Furthermore, I would like to restrict port access on
 my multi-port Service to a specific subset (e.g., application internal replication ports
 should not be shared, only user facing API). I would like to make the service visible and
 accessible in only the relevant clusters and namespaces where my application runs.

### User Story: Access to Cloud Resource

As an application owner, I would like to extend use of an **external** (i.e., non-k8s) Cloud Service
 to other clusters where my application runs. I would like to ensure the access is protected and
 limited only to workloads I manage in remote locations.

### User Story: Service Load Balancing and Failover

As an application owner, I would like to allow a Service I use to be provided by multiple remote
 locations. Selecting the remote endpoint should allow load balancing among the locations, with a
 specific cluster acting as the primary location but allowing failover to other locations when
 the primary is unavailable.

## Proposal and Design Details

### Namespaces and RBAC

ClusterLink shall be deployed into a designated namespace (e.g., `clusterlink-system` ),
 separate from user accessible namespaces, where workloads run. The use of a designated
 namespace allows the definition of roles and permissions needed to manage the system.
 The ClusterLink namespace holds objects used by the site administrator, including `Peer`
 and `AdminPolicy`, and has the relevant RBAC configuration needed. These object will
 only be watched for in the designated namespace and ignored in other namespaces (note:
 to allow future extension to multiple fabrics, the control plane shall watch for these
 objects in its local namespace and not a hard-coded namespace).

Application owners can define `Export`, `Import` and `Policy` objects in namespaces
 where their workloads run. In addition, application owners require read access to
 `Peer` objects in the ClusterLink namespace, as `Import`s need to reference a remote
 cluster by its name.

### Custom Resource Definitions

All ClusterLink CRD objects are derived from the current API structures defined, with
 changes and adaptations needed by CRD schemas. All objects are namespace scoped.
 Where possible, CRDs shall follow k8s recommendations for defining and managing
 objects (e.g., clear separation of Spec and Status fields, where Spec is set by the user
 and Status by ClusterLink controllers, use of sub-resources, etc).
 Spec parts declare user intent and the system's actual state is reconciled with
 the desired state asynchronously, over time. Thus, users should not expect immediate
 feedback and state change. Other relevant practices should be followed as much as possible
 (e.g., addition of Finalizers where appropriate, reconciliation retrieving up to date
 information instead of using internal caches for actual state, etc,).

Objects should use Conditions for tracking their Status. The following conventions for
 status conditions are suggested. Controllers should use the following conditions types
 to signal whether a resource has been reconciled, or if it encountered any problems:

- `Reconciling`: Indicates that the resource actual state does not yet match its desired
 state as defined by the Spec. A value of "True" means the controller is in the process
 of reconciling the resource. A "False" value indicates the controller has no work left.
- `Stalled`: Indicates that the controller is not able to make progress towards
 reconciling the resource (e.g., API error, taking longer than expected to reach the desired
 state, etc.). A "True" value for the condition should be interpreted that something
 might be wrong. It does not mean that the resource will never be reconciled.

The controller should also set the `observedGeneration` field in the CRD instances status
 every time it sees a new generation of the resource. This allows the controller (and users)
 to distinguish between resources that do not have any conditions set because they have been
 fully reconciled, from resources that have no conditions set because they have just been
 created and the controller did not have a chance to attempt reconciliation.

#### Peer CRD

The `Peer` object defines a remote cluster and its ClusterLink gateways. The local
 peer is not represented by a dedicated `Peer` CRD instance, but rather through a
 collection of other k8s objects such as deployment, secret, etc.

```go
type Peer struct {
  metav1.TypeMeta  
  metav1.ObjectMeta // Peer name must match the Subject name presented in its certificate

  Spec   PeerSpec
  Status PeerStatus
}

type PeerSpec struct {
  Gateways []string // one or more gateway addresses, each in the form "host:port"
  Attributes map[string]string // Peer's attribute set
  // TODO: should we have fixed/required set of attributes explicitly called out and
  // an optional attribute set encoded as a map?)
}
 
type PeerStatus struct {
  ObservedGeneration int64
  Conditions[] metav1.Condition
}
```

Peers are added to the ClusterLink namespace by the site administrator. Information
 regarding peer gateways and attributes is communicated out of band and not in scope
 for this design. Not having any gateways is an error but other than that there is
 no actual state for a Peer and the object can be reconciled immediately. Besides the
 recommended reconciliation condition types, a Peer should also support a
 `Reachable` (or `Seen`) condition indicating whether the peer is currently reachable,
 and the last time it successfully responded to heartbeats.

Peer names are unique and must align with the Subject name present in their certificate
 during connection establishment. The name is used by importers in referencing an export.

#### Export and Import

The `Export` object makes a local k8s Service endpoint potentially accessible to other
 clusters (if allowed if policy). The control plane must not allow access to services
 not explicitly exported. An `Export` can only refer to a Service in the same namespace
 as it but the Service type can be any of the supported k8s Services types (specifically
 allowing `ExternalName` Services). Each export defines a single port and exporting a
 multi-port Service requires multiple `Export`s defined.

```go
type Export struct {
  metav1.TypeMeta  
  metav1.ObjectMeta

  Spec   ExportSpec
  Status ExportStatus
}

type ExportSpec struct {
  Service string // name of exported service. If omitted, uses the Object name
                 // from the metadata. The exported Service must be defined and
                 // in the current namespace
  Port int16 // exported service port 
             // TODO: should we support named ports or only numbers?
             // TODO: Port can be omitted if the Service has a single port?
}

type ExportStatus struct {
  ObservedGeneration int64
  Conditions[] metav1.Condition
  // TODO: are any additional fields needed? For example, should we reflect the actual port here?
}
```

TODO: how and where are Service attributes defined? Both import and export are set
 by a user and thus are not trustworthy.

The Status should reflect the object's reconciliation status. If any changes are
 made to the target service, the export status is expected to update accordingly.
 Potential failure modes might include

- The target service does not exists.
- The target service port is missing or does not exist, or the port is unspecified
 and the Service has multiple ports.
- TODO: (optional) report the status of DNS lookup on `Spec.Service`?

In the case of a multi-port Service, the user must create an `Export` object for
 each exposed port they wish make available. All such objects must have the same
 `Spec.Service` value.

A remote Service is added to a namespace by defining an `Import` object. Note that
 unlike the current API design, `Binding` objects are no longer used/needed. Instead,
 the import defines an array of `Sources` providing the imported service.

```go
type Import struct {
  metav1.TypeMeta  
  metav1.ObjectMeta

  Spec   ImportSpec
  Status ImportStatus
}

type ImportSpec struct {
  Port int16 // Port exposed to the user
  TargetPort int16 // Target port open on the data plane Pods
     // TODO: should TargetPort be moved to the Status? User does not really set it
  Sources []string // Sources for the Service (e.g., in the form of "peer/namespace/export")
     // TODO: should we create a structure for this instead of using a string scheme?
     // TODO: is it reasonable to expect users to know remote values? Easier in central management...
     //       Alternative: define "Import" CRDs as catalog in the ClusterLink namespace and use object
     //       references in per namespace Import objects, acting as "binding" to the catalog?
     //       The application namespace Import Spec simply becomes a reference (name only suffices?)
     //       and this Spec moves to "Import" catalog objects.
  LoadBalancingPolicy struct{} // TODO: TBD (e.g., take example from Envoy's simpler policies).
}
 
type ImportStatus struct {
  ObservedGeneration int64
  Conditions[] metav1.Condition
  // TODO: are any additional fields needed? For example, TargetPort here?
}
```

Each `Import` object results in the creation of a local (to the namespace)
 Service and an additional "target" Service created in the dedicated ClusterLink
 namespace. The local Service is created of type `ExternalName` and has the name
 of the `Import`, as set by the user. It sole purpose is to refer to the target
 Service in the dedicated namespace. This required since k8s Services can only
 Select Pods in their own namespace. The target Service can be randomly named,
 but should be marked/annotate to allow back tracking to the user's `Import`.
 An alternative implementation could use a single Service backed by an `EndpointSlice`
 referring to the appropriate port open on ClusterLink data plane gateways.

The status of the import can reflect the following errors:

- one or more of the Sources are invalid or unreachable.
- the namespace already contains a k8s service which is not a derivation
 of the `Import` (i.e., no overwrites of user Services is allowed, only updates
 to `Import` objects).
- The k8s service for the import cannot be created.
 
#### Policy

TBD. The topic possibly requires its own design spec to address below questions.

- TODO: define Go objects (separate for Privileged and User?)
- TODO: encapsulate all actions (access, load balancing, etc.) on source and
 destination or separate types?
- TODO: ensure applies to source/destination in the same namespace only (k8s
 selects Pods in current namespace only, then defines ingress/egress. Using
 arbitrary workload sets makes this less clear).

### Control Plane Watchers and Reconciliations

The ClusterLink runtime configuration custom controller is responsible for
 watching and reacting to changes in the CRDs as well as built-in objects,
 as described below. While the control loops watch and react to events in order
 to achieve multiple goals, they may be combined into a single controller.
 Most of the actions can be a simple function call, delegating work to the
 policy engine and/or xDS manager, and should not result in
 overly complex reconciliation code. Logically, the controllers can be
 grouped as follows:

- **xDS** update events are generated in response to changes in `Peer`,
 `Export` and `Import` objects. They program the data plane (e.g., create
 xds.Listener for `Imports`, and xds.Cluster in response to `Peer` and
 `Export` objects, etc.)
- **Policy and Authz** management reacts to updates in `Peer`, `Export`, `Import`
 and `Policy` objects. In addition, the authorization requires that the controller
 maintain a mapping of (in-cluster) IP addresses to their respective Pods. The
 mapping is used to retrieve Pod attributes given the client IP observed by the
 data plane.
- **Control Plane** coordinates actions that require coordination. This is the
 only controller logic that requires leader election. The xDS and Policy/Authz
 control loops can be scaled as needed for HA/DR. Coordinated actions include,
 for example, updates to Peer status and heartbeats, assignment of data plane
 import ports to Services, etc.

## Impacts / Key Questions

The ClusterLink fabric is assumed to be single tenant in a cluster. That is, we deploy
 ClusterLink as a privileged/cluster scoped entity and do not currently support co-located
 fabrics in the same cluster. In the future we may want to support namespace-scoped
 deployments.

Our user model assumes three roles as described [above](#users). More granular roles might
 be needed in the future.

## Risks and Mitigations

- Focus and reliance on k8s based management could impact ClusterLink's interoperability
 in other environments, such as `docker` or physical/virtual machines. It may be possible
 to mitigate such risk by packaging the control plane to run over a lightweight k8s
 distribution, such as `k3s`. The control- to data-plane communication shall continue to
 use xDS, so the impact of k8s management could potentially be confined to the control plane
 only.

### Security Considerations

The current design improves security over what's currently supported. Use of k8s RBAC and
 maintaining dedicated namespaces for ClusterLink, would allow limiting exposure and risks.

- ClusteLink control and data planes deploy into a dedicated namespace (e.g.,
 `clusterlink-system`).
- only Site administrator should have privileges on that namespace.
- the control plane shall watch for Peers only in its own (dedicated) namespace. This
 is intentionally not hardcoded to allow flexibility in installation to alternative
 namespace in the future.
- application owners may define Imports and Exports in their own namespace.
- Exports must only relate to Services defined in the same namespace.
- TODO: differentiate the User and Admin policies based on namespaces as well?

### Testing Plan

<!-- An overview on the approaches used to test the implementation. -->
- Unit tests can be done using fake clients with well known state and changes.
- System tests can be run using the end-to-end testing framework, when unit tests provide
 insufficient testing coverage or excessive mocking.

### Update/Rollback Compatibility

<!-- How the design impacts update compatibility and how users can test rollout and rollback.-->

Implementing this design would require considerable change in code and documentation. Backward
 compatibility (e.g., with `gwctl`) is not planned. It may be possible to use an intermediate
 REST API server to support existing use cases that require it. Extensions to the CRD and their
 definitions shall be done using k8s standards (e.g., v1alpha, v1beta, v2alpha, etc.).

### Scalability

<!-- Describe how the design scales, especially how changes API calls, resource usage, or
 impacts SLI/SLOs.-->

TBD

### Implementation Phases/History

<!-- Describe the development and implementation phases planned to break up the work and/or
 record them here as they occur. Provide enough detail so readers may track the major
 milestones in the lifecycle of the design proposal and correlate them with issues, PRs,
 and releases occurring within the project.-->

TBD
