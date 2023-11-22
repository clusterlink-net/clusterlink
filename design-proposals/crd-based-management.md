# Design Proposal Template

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

<!-- Describe the need, problem, and motivation for the feature or change and any context
 required to understand the motivation. -->

Currently ClusterLink exposes a RESTful API for its users. A convenience CLI (`gwctl`)
 is provided for easier API access. The API/CLI is based on Created/Read/Update/Delete
 (CRUD) of the following objects:

- The `local ClusterLink peer` (is this explicit today?)
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

<!-- Describe any potential impact this feature or change would have. Readers should be able
 to understand why the feature or change is important. Briefly describe the desired
 outcome if the change or feature were implemented as designed. -->

TODO The reverse of some of the above...

### Prior discussion and links

<!-- Often these proposals start as an issue, forum post, email and it's helpful to link to
 those resources in this section to provide context and credit the right people for the
 idea.

It is vital for projects to be able to track the chain of custody for a proposed
 enhancement from conception through implementation which can sometimes be difficult to
 do in a single Github issue, especially when it is a larger design decision or cuts
 across multiple areas of the project.

The purpose of the design proposal processes is to reduce the amount of "siloed
 knowledge" in a community. By moving decisions from a smattering of mailing lists, video
 calls, slack messages, GitHub exchanges, and hallway conversations into a well tracked
 artifact, the process aims to enhance communication and discoverability. -->

TODO Some initial ideas in GH issue. Reference.

## (Optional) User/User Story

<!-- Define who your the intended users of the feature or change are. Detail the things that
 your users will be able to do with the feature if it is implemented. Include as much
 detail as possible so that people can understand the "how" of the system. This can also
 be combined with the prior sections. -->

### Users

Multiple users

- Fabric admin: control over entire fabric (e.g, create fabric, add peers to fabric);
 mostly not in scope for this design doc
- site admin: control single cluster. Add existing peers

### User story 1

### User story 2

... make sure to highlight need for permissions (e.g., application to policy only to 
 "my traffic", which persona can add/rm peer, exports/imports, ...)

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

## (Optional) Future Milestones

<!-- List things that the design will enable but that are out of scope for now. This can help
 understand the greater impact of a proposal without requiring to extend the scope of a
 proposal unnecessarily. -->

Consider moving some of the "out of scope" items here.
 Specifically, can discuss CM, unprivileged, automation / UX improvements such as
 export all services, etc.

## (Optional) Implementation Details

<!--
Some projects may desire to track the implementation details in the design proposal. Some
 sections may include:
-->

TODO: consider changing to "non functionals" and moving Security sections here as well.

### Testing Plan

<!-- An overview on the approaches used to test the implementation. -->

### Update/Rollback Compatibility

<!-- How the design impacts update compatibility and how users can test rollout and rollback.-->

I think it is out of scope for now, versus current state (need to change docs obviously...)
 Future changes based on k8s standards (e.g., v1alpha, v1beta, v2alpha, ...)

### Scalability

<!-- Describe how the design scales, especially how changes API calls, resource usage, or
 impacts SLI/SLOs.-->

### Implementation Phases/History

<!-- Describe the development and implementation phases planned to break up the work and/or
 record them here as they occur. Provide enough detail so readers may track the major
 milestones in the lifecycle of the design proposal and correlate them with issues, PRs,
 and releases occurring within the project.-->
