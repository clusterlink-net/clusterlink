---
title: About ClusterLink
linkTitle: About
description: Why and where should you use ClusterLink
menu: {main: {weight: 30, pre: "<i class='fa-solid fa-info-circle'></i>" }}
---

{{% blocks/section color="secondary" %}}

## About ClusterLink

{{% pageinfo %}}
ClusterLink is in **alpha** status and not ready for production use.
{{% /pageinfo %}}

ClusterLink is an [open source][] project that offers a secure and performant
 solution for interconnecting services across multiple clusters in
 different domains, networks, and cloud infrastructures.

 Compared with other solutions in this space, ClusterLink control plane is simpler
 to configure and operate at scale, and its data plane is more secure, scalable and
 efficient.

ClusterLink's management model is built on the following abstractions:

- **Fabric**: a set of collaborating clusters, all sharing the same root of trust.
- **Peer**: a specific cluster in a fabric. Each fabric peer makes independent
 decisions on service sharing and access control.
- **Export**/**Import**: services must be explicitly shared by clusters before
 they can be used.
- **Access policies**: ClusterLink supports fine-grained segmentation with a
 "default deny" policy, adhering to "zero trust" principles.

## Architecture

ClusterLink consists of several main components that work together to securely
 connect workloads across multiple Kubernetes clusters, both on-premises and on
 public clouds. These run as regular Kubernetes deployments and can take advantage
 of existing mechanisms such as horizontal scaling, rolling deployments, etc.

ClusterLink uses the Kubernetes API servers for its configuration store. The
 **control plane** is responsible for watching for changes in relevant built-in
 and custom resources and configuring the data plane Pods.
 The control plane is also responsible for managing local Kubernetes
 services and endpoints corresponding to imported remote services. By using
 standard Kubernetes services, ClusterLink integrates seamlessly with the Kubernetes
 network model, and can work with any Kubernetes distribution, CNI and IP address
 management scheme.

The local service endpoints refer to **data plane** Pods, responsible for
 workload-to-service secure tunnels to other clusters. The data plane uses
 [HTTP CONNECT][] with [mutual TLS][] for security.
 The use of HTTPS over tcp/443 removes the need for VPNs and special firewall
 configurations. Certificate-based mTLS guarantees in-transit data
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

- **clusters are aligned with organizational units** - sharing *internal*
 microservices with other clusters and namespaces should be limited to those
 belonging to the same unit, while also communicating with *exposed* services
 belonging to other units.
- services are owned by **different administrative domains** (e.g., different
 development teams) - thus judicious sharing and more stringent access
 controls across clusters are needed.
- it is desirable to **increase scalability and limit information sharing** by
 minimizing information exchanged between clusters - with ClusterLink, each
 cluster manages its own naming and load balancing, requiring considerable
 less cross-cluster metadata for its communication.
- there is a need for **separation of concerns** - that is, between network administrators
 and application owners.

{{% /blocks/section %}}

[open source]: https://github.com/clusterlink-net/clusterlink
[HTTP CONNECT]: https://en.wikipedia.org/wiki/HTTP_tunnel
[mutual TLS]: https://en.wikipedia.org/wiki/Mutual_authentication#mTLS
