# Design Proposal: Policy Attributes

**Authors**: @zivnevo, @elevran

**Begin Design Discussion**: 2023-11-20

**Status:** draft

## Summary/Abstract

ClusterLink policies apply to communications between workloads.
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

- **trustworthiness**, related the level of trust we can place in its derivation (e.g.,
 complexity and skill required in affecting the attribute's value). Ideally, policies make
 judicious use of attributes based on the level of trust and sensitivity of the workloads
 in communication.
- **usefulness**, relating to the amount of unique context provided by the attribute.

### Impact and desired outcome

The current set of policy attributes is incomplete and not well defined.
 This leaves 

Describe any potential impact this feature or change would have. Readers should be able
 to understand why the feature or change is important. Briefly describe the desired
 outcome if the change or feature were implemented as designed.

### Prior discussion and links

Not applicable

## (Optional) User/User Story

Define who your the intended users of the feature or change are. Detail the things that
 your users will be able to do with the feature if it is implemented. Include as much
 detail as possible so that people can understand the "how" of the system. This can also
 be combined with the prior sections.

 As a (site/fabric) administrator I would like to define a (access control/load balancing)
  policy based on the source and destination attributes.
 
 trust attributes

 give example use cases for policies using attributes (access in 3-tier, stay in EU, load balancing).

## Goals

* Define the set
* Flexible enough to ...
* Strict enough to limit applicability

List the desired goal or goals that the design is intended to achieve. These goals can be
 leveraged to structure and scope the design and discussions and may be reused as the
 "definition of done" -  the criteria you use to know the implementation of the design
 has succeeded in accomplishing its goals.

## Non-Goals

Describe what is out of scope for the design proposal. Listing non-goals helps to focus
 discussion and make progress.

* Non k8s environments.
* Extensible
* formal attestation
* backward compatible

## Proposal

The proposal will address following aspects:

- workload attributes
- service attributes (fixed set or augmented by user)
- fabric/site attributes (fixed set or augmented by user)
- attributes are scoped (e.g., "cl-site:geo", "k8s:ns:, not "geo","ns")
- attributes are not typed (strings only)
- trustworthiness varies (let the user / policy write decide what to set in policy)
- the actual set itself needs to be defined
- when and how exchanged to allow policy decisions

### Retrieving attributes

If we assume the following are true:

- K8s apiserver can be trusted
- authentication/authorization is corrected configured on the K8s api server

The following attributes can be used to uniquely define a workload

- K8s namespace
- K8s labels

As users are isolated in their own namespaces, it is not possible for an attacker to provision resources in arbitrary namespaces and impersonate another workload. Labels, then, are used to differentiate the different workloads within the namespace. Assuming they are configured correctly by the workload owner, this should be sufficient to uniquely specify workloads safely.


This is where we get down to the specifics of what the proposal actually is. It should
 have enough detail that reviewers can understand exactly what you're proposing, but
 should not include things like API designs or implementation. This section should expand
 on the desired outcome and include details on how to measure success.

Future: add other attestors.

## Impacts / Key Questions

List crucial impacts and key questions, some of which may still be open. They likely
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

Ziv has many ;-\

## Future Milestones

List things that the design will enable but that are out of scope for now. This can help
 understand the greater impact of a proposal without requiring to extend the scope of a
 proposal unnecessarily.

Additional attribute sources in the future
Additional attributes
User flexibility (e.g., for service and site  whenever we can learn something from user via API)

## Non Functional

### Testing Plan

An overview on the approaches used to test the implementation.

### Update/Rollback Compatibility

How the design impacts update compatibility and how users can test rollout and rollback.

new and old version?

### Scalability

Describe how the design scales, especially how changes API calls, resource usage, or
 impacts SLI/SLOs.

### Security Considerations

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

### Implementation Phases/History

Describe the development and implementation phases planned to break up the work and/or
 record them here as they occur. Provide enough detail so readers may track the major
 milestones in the lifecycle of the design proposal and correlate them with issues, PRs,
 and releases occurring within the project.

gateway attr (encoded in cert?)
workload attr, collected by control plane
service attr, defined by user, carried over on import