---
title: Overview
weight: 10
---


## What is ClusterLink?

<!-- Introduce your project, including what it does or lets you do, why you would use it, and its primary goal (and how it achieves it). This should be similar to your README description, though you can go into a little more detail here if you want. -->
ClusterLink simplifies the connection between application services that are located in different domains,
 networks, and cloud infrastructures.

## When should I use it?

<!-- Help your user know if your project will help them. Useful information can include: 

* **What is it good for?**: What types of problems does ClusterLink solve? What are the benefits of using it?

* **What is it not good for?**: For example, point out situations that might intuitively seem suited for your project, but aren't for some reason. Also mention known limitations, scaling issues, or anything else that might let your users know if the project is not for them.

* **What is it *not yet* good for?**: Highlight any useful features that are coming soon.
-->

ClusterLink is useful when multiple parties are collaborating across administrative boundaries.
 With ClusterLink, information sharing policies can be defined, customized, and programmatically
 accessed around the world by the right people for maximum productivity while optimizing network
 performance and security.

## How does it work?

ClusterLink uses a set of unprivileged gateways serving connections to and from K8s services according to policies
 defined through the management APIs. ClusterLink gateways establish mTLS connections between them and
 continuously exchange control-plane information, forming a secure distributed control plane.
 In addition, ClusterLink gateways represent the remotely deployed services to applications running in a local cluster,
 acting as L4 proxies. On connection establishment, the control plane components in the source and the target ClusterLink
 gateways validate and establish the connection based on specified policies, then promote the control connection into a
 data plane session, with no overhead.

## Why is it unique?

The distributed control plane and the fine grained connection establishment control are the main
 advantages of ClusterLink over some of its competitors. Performance evaluation on clusters deployed in the same
 Google Cloud zone shows that ClusterLink can outperform some existing solutions by almost 2× while providing
 fine grained authorization on a per connection basis.

## Where should I go next?

* [Getting Started]({{< ref "getting-started" >}}): Get started with ClusterLink
* [Tutorials]({{< ref "tutorials" >}}): Check out some examples and step-by-step
  instructions for different use cases.
