---
title: "Introducing ClusterLink: Simplifying Multicluster Service Connectivity"
linkTitle: Introducing ClusterLink
date: 2024-04-22
author: Etai Lev Ran
type: blog
draft: true
---

{{% imgproc "field" Fill "800x250" /%}}

Deploying microservice applications across multiple clusters offers many benefits.
 These include, for example, improved fault tolerance, increased scalability, performance
 or regulatory compliance. In addition, use of multiple locations may be required to
 access specialized hardware, services or data sources available only in those
 locations. Kubernetes SIG multicluster [lists][SIG-MC] additional reasons for
 using multiple clusters.

When attempting to realize the benefits of multicluster applications, at least two
 aspects must be addressed: orchestration (sometimes referred to as *"scheduling"*)
 and connectivity. There are several existing open source projects providing these
 capabilities. For example, projects such as [Open Cluster Management][OCM]
 and [KubeStellar][] simplify orchestration and management of
 workloads across multiple clusters.

In order to facilitate cross cluster communications, you could use Kubernetes
 native resources, including [Ingress][] and [Gateway API][]
 Or, alternatively, choose from a number of existing open source projects, such
 as [Istio][], [Skupper][] and [Submariner][].
 By and large, these solutions attempt to conjoin the multiple clusters, flattening the
 isolated networks into a single flat "mesh" shared between the connected clusters.
 The goal is, to the extent possible, extend the Kubernetes single cluster network
 abstraction to multicluster use cases.

The creation of a shared mesh is not always desirable and may place additional
 constraints on administrators, developers and the workloads they manage. For example,
 it might make assumptions on objects, such as Services, being the "same" based
 on the objects sharing a name (i.e., namespace sameness as defined [by Istio][]
 and [by SIG-MC][]), or require planning IP addresses assignment across independent clusters.

This post introduces [ClusterLink][], an [open source][] project that
 offers a different design point in the multicluster networking solution space.
 We believe it provides a solution that is simpler to configure and operate, and
 offers more secure, scalable and performant control and data planes for
 multicluster service connectivity.

## Introducing ClusterLink

ClusterLink offers a secure and performant solution to interconnect services
 across multiple clusters. It has a simple management model, built on the following
 abstractions:

- **Fabric**: a set of collaborating clusters, all sharing the same root of trust.
 Clusters must be part of a fabric to enable multicluster networking.
- **Peer**: a specific cluster in a fabric. Each peer is identified by a certificate,
 signed by the fabric's certificate authority. Each peer makes independent
 decisions on service sharing and access control.
- **Export**/**Import**: services must be explicitly shared by clusters before
 they can be used. A service can be imported by any number of peers. To increase
 availability or performance, a service can be exported by more than one peer.
- **Access policies**: ClusterLink supports fine grained segmentation with a
 "default deny" policy, adhering to "zero trust" principles. Access policies are
 used to explicitly allow and deny communications. Affected workloads are defined
 in terms of their attributes (such as location, environment, namespace or even
 labels) and have two priorities, with privileged (i.e., administrator defined,
 cluster scoped) policies evaluated before user-defined namespaced policies.

ClusterLink consists of several main components that work together to securely
 connect workloads across multiple Kubernetes clusters, both on-premises and on
 public clouds. These run as regular Kubernetes deployments and can take advantage
 of existing mechanisms such as horizontal scaling, rolling deployments, etc.

ClusterLink uses the Kubernetes API servers for its configuration store. The
 **control plane** is responsible for watching for changes in relevant built-in
 and custom resources and configuring the data plane Pod using [Envoy's xDS][] protocol.
 The control plane is also responsible for managing local Kubernetes
 services and endpoints corresponding to imported remote services. By using
 standard Kubernetes services, ClusterLink integrates seamlessly with the Kubernetes
 network model, and can work with any Kubernetes distribution, CNI and IP address
 management scheme.

The local service endpoints refer to **data plane** Pods, responsible for
 workload-to-service secure tunnels to other clusters. The data plane uses
 [HTTP CONNECT][] with [mutual TLS][] for security.
 The use of HTTPS over tcp/443 removes the need for VPNs and special firewall
 configurations. Certificate based mTLS guarantees in-transit data
 encryption and limits allowed connections to other fabric peers only. In addition,
 all data plane connections between clusters are explicitly approved by the
 control plane and must pass independent egress and ingress access policies
 before any workload data is carried across.

### Use cases

In addition to the typical multicluster networking use cases, such as
 HA/DR, cloud bursting, and connecting microservices deployed across
 geographically distributed clusters, ClusterLink can also provide
 significant benefits which are not well served by other solutions.
 Specifically, ClusterLink can address requirements of use cases where:

- **clusters are aligned with organizational units** and sharing *internal*
 microservices with other clusters and namespaces should be limited to those
 belonging to the same unit, while also communicating with *exposed* services
 belonging to other units.
- services are owned by **different administrative domains** (e.g., different
 development teams) and thus judicious sharing and more stringent access
 controls across clusters are needed.
- it is desirable to **increase scalability and limit information sharing** by
 minimizing information exchanged between clusters. With ClusterLink, each
 cluster manages its own naming and load balancing, requiring considerable
 less cross-cluster metadata for its communication.
- there is a need for **separation of concerns** between network administrators
 and application owners.

## Getting started with ClusterLink

To get started with ClusterLink, we invite you to explore the rest of the
 documentation and familiarize yourself with its concepts and operation.
 When ready, try out the [getting started tutorial][].

We would love to [hear feedback][] and explore how we can make ClusterLink better.
 ClusterLink is an open source project under the Apache license, and We encourage
 contributions such as [opening issues or enhancement requests][], submitting
 [pull requests][], or contributing documentation.

If you think there's a use case where ClusterLink can help in simplifying
 multicluster service connectivity and improve security for your multicluster
 deployments, we'd love to collaborate!

[SIG-MC]: https://multicluster.sigs.k8s.io/#problem-statement-why-multicluster
[OCM]: https://open-cluster-management.io/
[KubeStellar]: https://kubestellar.io
[Ingress]: https://kubernetes.io/docs/concepts/services-networking/ingress/
[Gateway API]: https://kubernetes.io/docs/concepts/services-networking/gateway/
[Istio]: https://istio.io
[Skupper]: https://skupper.io
[Submariner]: https://submariner.io/
[by Istio]: https://istio.io/latest/docs/ops/configuration/traffic-management/multicluster/
[by SIG-MC]: https://github.com/kubernetes/community/blob/master/sig-multicluster/namespace-sameness-position-statement.md
[ClusterLink]: https://clusterlink.net
[open source]: https://github.com/clusterlink-net/clusterlink
[Envoy's xDS]: https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/operations/dynamic_configuration
[HTTP CONNECT]: https://en.wikipedia.org/wiki/HTTP_tunnel
[mutual TLS]: https://en.wikipedia.org/wiki/Mutual_authentication#mTLS
[hear feedback]: https://groups.google.com/g/clusterlink-users
[opening issues or enhancement requests]: https://github.com/clusterlink-net/clusterlink/issues
[pull requests]: https://github.com/clusterlink-net/clusterlink/pulls
[getting started tutorial]: /docs/{{< param latest_stable_version >}}/tutorials/iperf
