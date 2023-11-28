# Design Proposal: Policy Attributes

**Authors**: @zivnevo, @elevran

**Begin Design Discussion**: 2023-11-20

**Status:** draft

## Summary/Abstract

ClusterLink policies apply to communications between workloads. [ZN: should we mention services here?]
 Workloads can be identified by a strong (e.g., cryptographic) identity.
 The identity links the workload to a set of attributes, and policies are
 defined on workload attributes. This design proposal defines the initial
 set of attributes used in policies.

## Background

### Motivation and problem space

ClusterLink exchanges workload attributes when determining policies governing communications.
 Policies affect different communication aspects including, for example, access control and
 load balancing. ClusterLink gateways serve as enforcement points for egress and ingress traffic.
 The set of attributes is ill defined. We would like to define the set of attributes used
 in exchange and policies.

The set of attributes applicable to a communication flow is either determined by the control
 plane at runtime or derived from the workload's identity document. We can associate two measures
 with each attribute:

- **trustworthiness**, relating the level of trust we can place in its derivation (e.g., permission-level,
 complexity and skill required in affecting the attribute's value). Ideally, policies make
 judicious use of attributes based on the level of trust and sensitivity of the communicating workloads.
- **usefulness**, relating to the amount of unique context provided by the attribute.
 For example, attributes that set the workload's application tier are far more useful than
 arbitrarily-set attributes such as process id or creation timestamp.

### Impact and desired outcome

The current set of policy attributes is incomplete and not well defined.
 This leaves the implementation to make decisions that are not fully transparent to users.
 Defining the (initial) set of attributes used in policies, would allow ClusterLink users to
 make informed and stable decisions about policy definition as suited for their use case and
 requirements.

### Prior discussion and links

Not applicable.

## User/User Story

- **Access control based on cluster geography**: As a network administrator I would like to enable
 Service access only from certain locations (e.g., EU only, to comply with GDPR).
- **Load balancing based on cluster geography**: As a network administrator I would like to set
 a policy that, when a Service is provided by several remote locations, only locations with the
 same geography as the source should be considered.
- **Access control based on cluster identity**: As a Service owner, I would like to allow
 access to a specific service only from another clusters I own.
- **Access control based on workload namespace and labels**: as a Service owner, I would
 like to enable enable access to a service based on the source workload namespace and its "role" label
 value, regardless of cluster where the workload is running (e.g., assumes clusters are used as the
 infrastructure and teams are allocated the same namespace across all clusters).
 Labels of workloads running in other namespaces are not trusted.

## Goals

This design document should:

- Define the (initial) set of attributes available for policy definition and enforcement. The set
 may be extended in the future.
- Define the source of each attribute (i.e., where retrieved), along with some assessment of its
 trustworthiness and usefulness.
- Define how attributes are encoded in policy definition and exchanged in gateway communications
 to enable policy enforcement.

## Non-Goals

The following aspects are explicitly excluded and are out of scope in the current design:

- Defining policy attributes and their facets in environments other than Kubernetes.
- Define the life-cycle management of the attribute set (i.e., how attributes are added, deprecated
 and modified in a backward compatible manner).
- Define the process of formal and provable attestation of attributes and their values. This topic
 is partially addressed by assigning different trustworthiness measures to different attributes.

## Proposal

We propose to have attributes defined at different scope/layer, with each object implicitly assigned
 attributes of containing layer:

- Site level attributes (either a fixed set defined by ClusterLink or extended by user according to
 the Fabric configuration [1]) that are pertinent to all workloads and Services in the site. Examples
 may include `geography`, `cloud-provider`, `cloud-region`, `cluster-name`, etc.
- Service level attributes (either a fixed set or augmented by user in the fabric configuration).
 These may includes such attributes as `service-name`, `namespace`, `labels`, etc. Other attributes may
 be derived from the Kubernetes Service definition, if relevant. Services are assigned the Site
 attributes as well.
- Workload attributes are associated with a specific workload instance, and may include, for example,
 `service-account`, `namespace`, `image-name`, etc. Workloads are assigned the Site attributes as well.

[1] Fabric level configuration could be used to define the set of attributes that can be defined per Site.
 The concept of a fabric defines a "container" for sites that can potentially communicate with each other.
 The fabric defines the root of trust as well as any global configuration.

### General Properties of Attributes

- All attributes are key-value pairs. Keys are unique within a set (i.e., can't appear more than once).
- Attributes are scoped. Scope is set in the key prefix (e.g., "cl-site:geo", "k8s:ns:, not "geo","ns").
 This potentially enables future extension to other environments without having to overload concepts.
- Attributes are not typed - the value in the key-value pair is always a string. This enables the use
 of match expressions (e.g., *is*, *is not*, *is one of*, etc.).
- Attribute trustworthiness varies. The user / policy writer is ultimately responsible for deciding
 what attributes are relevant in a policy.

### Workload Attributes

If we assume the following are true:

- Replies from Kubernetes API server can be trusted;
- authentication/authorization is correctly configured on the Kubernetes API server; and
- users are isolated in their own namespaces

then the following attributes can be used to identify a workload within a Site:

- K8s namespace
- K8s labels

As users are isolated in their own namespaces, it is not possible for an attacker to provision
 resources in arbitrary namespaces and impersonate another workload. Labels, then, are used to
 differentiate between the different workloads within the namespace. Assuming they are configured
 correctly by the workload owner, this should be sufficient to uniquely specify workloads safely.

### Service Attributes

Service attributes are set (or retrieved) when a Service is exported. Remote gateways become aware
 of the Service attributes when a service is first imported. If multiple bindings exist for an Import,
 All bound Services must have fully matching attribute set. A binding is declined when there is a
 mismatch between a first and later binding. Ideally, the management layer will ensure all gateways
 importing the same service, will see an identical set of attributes. this also favors that Services
 and Service attributes are set by the user in a central place and get distributed via management layer.
 The exact definition is out of scope of this design.

### Gateway Attributes

> ZN: Is this part of the gateway's certificate? How do we set it?

Gateways learn the attributes associated with other gateways when Peers are added.

### Attribute Table

| Attribute name | Scope | Source | Description | Comments |
| ---- | ----- | ------ | ----------- | -------- |
| `cl:fabric` |  Site/Fabric | configuration | fabric the site belongs to | Implicit via CA, might be useful in future for cross fabric communication |
| `site:name` | Site | configuration | site name | Configured when site is created |
| `site:location` | Site | configuration | site location | hierarchical (e.g., `aws/us-east/vpc17`) or split to flat attributes (e.g., `site:provider`, `site:region` - similar to `site:name`) |
| `site:environment` | Site | configuration | site environment (e.g., production, staging) | mandated or recommended? |
| `cl:site:<attr>` | Site | configuration | user defined site attributes | do we want to support these initially? |
| `cl:service:<attr>` | Service | configuration | user defined Service attributes | do we want to support these initially? |
| `service:name` | Service | k8s API (or Export/Import?) | Service name | is there a corresponding workload name? For workloads name are randomized, but the name of the "owner object" might be useful? |
| `k8s:ns` | Workload, Service | k8s API | Kubernetes namespace | |
| `k8s:label:<name>` | Workload, Service | k8s API | Kubernetes label(s) | the use of standard k8s labels is recommended. Labels describing the application structure (e.g., `app`, `role`, `tier`) could be expressive and flexible |
| `k8s:container-image` | Workload | k8s API | Image name | includes repo? Tag? Image SHA only?  what's useful? |
| `TBD` | TBD | TBD | TODO | ... |

### Exchanging Attributes Between Gateways

We would like to minimize handshake iterations on every connection request. To achieve that, all
 gateways keep the attributes of all other gateways. Moreover, all gateways keep the attributes
 of all imported/exported services. This leaves only the workload attributes to be transferred
 during a connection request, as detailed below.

> ELR: is the above needed? It is a bandwidth optimization (tradeoff state for lower bandwidth). Might
 a compression based solution suffice?

> ZN: management layer will need a mechanism to allow updating the attributes of gateways/services
 across the mesh.

**Client Side:**

1. The local gateway data plane gets a request from a local workload to connect to a remote service.
 The client handle and destination service are passed to the control plane.
1. The control plane extracts workload attributes from the cluster's API server. The client's IP address
 is used as a handle to identify the workload.
1. The control plane merges these attributes with its own (gateway attributes) to form the set of
 source attributes.
1. The control plane forms a collection of destination attribute sets, one set per remote-service binding.
 Each set of destination attributes contains both the attributes of the remote service and the attributes
 of the remote gateway exposing this service.
1. The control plane can now call the policy engine component with the set of source attributes
 and with the collection of sets of destination attributes.
1. The access-policy engine will filter down to the set of remote gateways that are allowed to provide the
 service (if any) based on access control policies set. The load-balancing-policy engine will choose one
 remote gateway out of this set based on the load balancing policies defined.
1. The selected destination will be returned to the data plane (potentially along with other configuration
 if needed), which can then initiate a connection request to the remote gateway.

**Server Side:**

1. The gateway on the cluster of the exported service gets a connection request from the client-side
 gateway. The connection request includes the attributes of the requesting workload.
1. The server-side gateway merges these attributes with the attributes of the client-side gateway
 to form the set of source attributes (note that the source site attributes are not sent to conserve
 resources - see note [here](#exchanging-attributes-between-gateways)).
1. The server-side gateway then merges the attributes of the requested service with its own set of gateway
 attributes to form the set of destination attributes.
1. It can now call the policy engine with the two sets of attributes and get an allow/deny answer.

## Impacts / Key Questions

<!-- List crucial impacts and key questions, some of which may still be open. They likely
 require discussion and are required to understand the trade-offs of the design. During
 the lifecycle of a design proposal, discussion on design aspects can be moved into this
 section. After reading through this section, it should be possible to understand any
 potentially negative or controversial impact of the design. It should also be possible
 to derive the key design questions: X vs Y. -->

<!-- This will also help people understand the caveats to the proposal, other important
 details that didn't come across above, and alternatives that could be considered. It can
 also be a good place to talk about core concepts and how they relate. It can be helpful
 to explicitly list the pros and cons of each decision. Later, this information can be
 reused to update project documentation, guides, and Frequently Asked Questions (FAQs).
-->

> Ziv has many ;-\

## Future Milestones

The design will enable the following which are out of scope for now:

- Support for additional attribute sources in the future
- Additional for additional attributes
- Adding and enforcing the setting of user defined attributes for services and sites

## Non Functional

### Testing Plan

TODO

### Update/Rollback Compatibility

We don't support backward compatibility. All policies and implemenration must be updated to the
 adhere to the specification defined by this design.

### Scalability

TODO: not applicable.

### Security Considerations

The introduction of ClusterLink gateways to a cluster, increases the 'surface area' exposed
 for attack, by allowing remote access to Services.

The following security considerations are impacted (though not necessarily directly by this design
 change which is more concerned with formalizing existing implementation):

- ClusterLink gateways are configured to establish mutually authenticated connections only with
 other gateways in the same Fabric (trust domain, certificate authority). This should limit
 some of the exposure.
- ClusterLink requires elevated permissions to read Pod and Service specification and status
 across multiple namespaces.
- The "trustworthiness" of attributes is paramount for effective access control.
- Similarly, the correctness of the policy engine impacts the operation and cross site
 communication.
- Users may opt-out of ClusterLink access by (1) not importing/exporting Services; (2) ensuring
 strict, default deny, policies are defined; and (3) potentially further locking down access by
 setting appropriate k8s NetworkPolicies on their sensitive Pods, disallowing access from the
 clusterLink namespace.

### Implementation Phases/History

<!-- Describe the development and implementation phases planned to break up the work and/or
 record them here as they occur. Provide enough detail so readers may track the major
 milestones in the lifecycle of the design proposal and correlate them with issues, PRs,
 and releases occurring within the project. -->

TODO

- gateway attr (encoded in cert?)
- workload attr, collected by control plane
- service attr, defined by user, carried over on import
