---
title: Hello World
linkTitle: hello, world!
date: 2024-03-24
author: Etai Lev Ran
description: >-
     Welcome to ClusterLink
type: blog
draft: true
---

Ah, hello - anyone there?

I'm new here and wanted to say hi and formally introduce myself.

Hi - I'm ClusterLink, and I'm the new kid on the block - just starting out as open-source
 project trying to find my place in the big, wide world of multicluster Kubernetes connectivity.
 This is my first venture away from my [home on GitHub](https://github.com/clusterlink-net/clusterlink)

At my core, I offer a simple yet powerful philosophy which I like to think of the three S'es:
 **Seamless**, **Simple**, and **Secure** Service Interconnect. Maybe it should have been the
 four S'es? Those off by one errors were always pesky... Anyway, where was I? Ah - yes:
 the three S'es.

## Seamless

One of the things I'm most proud of is my versatility. I work seamlessly with any Kubernetes
 distribution – whether it's managed or self-hosted, free or paid, on cloud or on-premise –
 and any Container Network Interface (CNI). Services exported from one cluster can be
 imported into any other cluster in the interconnect fabric and would appear to clients
 as a local Kubernetes Service. No need to worry about names and IP address overlaps, these are
 all private to each cluster.

## Simple

Exposing a service from one cluster to another is as simple as marking it as exported
 in the source cluster and defining an endpoint for it in the other. Each cluster
 retain control over its local load balancing decisions, and each cluster can use
 independent names and local administrative control. No assumptions here, no automatically
 merging independent locations and Services by name, no exchanging private Pod
 information all over the place – I like to keep things neat and tidy.

## Secure

Let me tell you, I take security very seriously:

- All cross-cluster communications are authenticated using certificates and mutual TLS.
- Services need to be explicitly exported and imported to be accessible across
 clusters - no accidental sharing just because clusters are joined into the same fabric
 or decide to collaborate on just one service.
- All communications attempts are subject to independent egress and ingress policies.
 By default, I operate on a "deny" basis, meaning that only explicitly allowed connections
 go through. Safety and control go hand in hand, after all.

And here's the icing on the cake: I believe in clear separation of concerns. That means
 network administrators and application owners each have their own piece of the pie
 when it comes to controlling ClusterLink configurations. Network administrator policies
 and configurations take precedence, so application owners stay "within bounds".
 No confusion, no fuss.

## So, what's next?

I'm super excited to announce that I'm being released as version 0.2.0.
 It's a big milestone for me, and I'm hoping to get some feedback and
 requirements from all of you lovely folks out there. After all, I'm here
 to evolve and grow just like everyone else.

First things first, though – I want to make it clear that I'm not quite
 ready for the big leagues yet. As an alpha project, I'm not suitable for
 production use. Basic functionality is working as far as I can tell, but
 I'm likely missing some features, and there might be a few bugs here and there.
 But hey, Rome wasn't built in a day either, right?

So, what do you say? Want to give me a try? I promise I won't disappoint.
 And if you find me useful (which I'm sure you will), there are plenty of
 ways you can help me grow and improve: join the [users' mailing list](https://groups.google.com/g/clusterlink-users)
 open [issues or enhancement requests](https://github.com/clusterlink-net/clusterlink/issues),
 provide additional [documentation](https://github.com/clusterlink-net/clusterlink/tree/main/website)
 or [code](https://github.com/clusterlink-net/clusterlink) – the possibilities are endless!

I can't wait to start on this journey with all of you. Together, we'll make
 the world of Kubernetes a better, safer, and more connected place.
 Happy cluster linking! 🚀
