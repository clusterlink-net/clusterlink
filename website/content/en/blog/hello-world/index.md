---
title: Hello, World!
linkTitle: hello, world!
date: 2024-04-02
author: Etai Lev Ran
type: blog
---

{{% imgproc "sunflower" Fill "800x250" /%}}

Hi everyone!

I’m ClusterLink, and as the new kid on the block, I’d like to
 say hello and introduce myself. I’m just starting out as an open-source
 project trying to find my place in the big, wide world of multicluster
 Kubernetes connectivity. This is my first venture away from
 my [home on GitHub](https://github.com/clusterlink-net/clusterlink).

My core is centered around three key principles: **Seamlessness**, **Simplicity**,
 and **Security**. I focus on applying these principles to service access across
 multiple clusters.

## Seamlessness

One of the things I’m most proud of is my versatility. I work seamlessly with any
 Kubernetes distribution, whether it’s managed or self-hosted, free or paid, on-cloud
 or on-premise, and any Container Network Interface (CNI). Services exported from one
 cluster can be imported into any other cluster in the ClusterLink fabric and appear
 to clients as a local Kubernetes Service. No need to worry about names and IP address
 overlaps, these are all private to each cluster.

## Simplicity

I like to keep things neat and tidy. Exposing a service from one cluster to another
 is as simple as marking it as exported in the source cluster and defining an imported
 endpoint for it in the other. Each cluster retains control over its local load balancing
 decisions, and each cluster can use independent names and local administrative control.
 No assumptions here, no automatically merging independent locations and services by name,
 and no exchanging private Pod information all over the place.

## Security

I take security very seriously:

- All cross-cluster communications are authenticated using certificates and mutual TLS.
- Services need to be explicitly exported and imported to be accessible across
 clusters - no accidental sharing just because clusters are joined into the same fabric
 or decide to collaborate on just one service.
- All communications attempts are subject to independent egress and ingress policies.
 By default, I operate on a “deny” basis, meaning that only explicitly allowed connections
 go through. Safety and control go hand in hand, after all.

And here’s the icing on the cake: I believe in clear separation of concerns. That means
 network administrators and application owners each have their own piece of the pie
 when it comes to controlling ClusterLink configurations. Network administrator policies
 and configurations take precedence, so application owners stay “within bounds”.
 No confusion, no fuss.

## So, what's next?

I’m super excited to announce that I’m being released as version 0.1.0.
 It’s a big milestone for me, and I’m hoping to get some feedback and
 requirements from all of you folks out there. After all, I’m here
 to evolve and grow just like everyone else.

First things first, though – I want to make it clear that I’m "work in progress".
 As an alpha stage project, I might not be suitable for production use yet. Basic
 functionality is working as far as I can tell, but I’m likely missing some features,
 and there might be a few bugs here and there. But hey, Rome wasn’t built in a day
 either, right?

So, what do you say? Want to give me a try? I promise I won’t disappoint.
 And if you find me useful (which I’m sure you will), there are plenty of
 ways you can help me grow and improve: join the [users' mailing list](https://groups.google.com/g/clusterlink-users),
 [issues or enhancement requests](https://github.com/clusterlink-net/clusterlink/issues),
 provide additional [documentation](https://github.com/clusterlink-net/clusterlink/tree/main/website)
 and [code](https://github.com/clusterlink-net/clusterlink), or make a suggestion.
 The possibilities are endless!

I can't wait to start on this journey with all of you. Together, we'll make
 the world of Kubernetes a better, safer, and more connected place.
 Happy cluster linking! 🚀
