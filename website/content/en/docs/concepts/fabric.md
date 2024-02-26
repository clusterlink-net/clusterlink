---
title: Fabric
description: Defining a ClusterLink fabric
weight: 32
---

<!--
Each task should give the user

* The prerequisites for this task, if any (this can be specified at the top of a multi-task page if they're the same for all the page's tasks. "All these tasks assume that you understand....and that you have already....").
* What this task accomplishes.
* Instructions for the task. If it involves editing a file, running a command, or writing code, provide code-formatted example snippets to show the user what to do! If there are multiple steps, provide them as a numbered list.
* If appropriate, links to related concept, tutorial, or example pages.
-->

The concept of a `Fabric` encapsulates a set of cooperating [sites](./sites.md).
 All sites in a fabric and communicate and may share [services](./services.md)
 between them, with access governed by [policies](./policies.md).
 The `Fabric` acts as a root of trust for site to site communications (i.e.,
 it functions as the certificate authority enabling mutual authentication between
 sites).

Currently, the concept of a `Fabric` is just that - a concept. It is not represented
 or backed by any managed resource in a ClusterLink deployment. Once a `Fabric` is created,
 its only relevance is in providing a certificate for use by each site's gateways.
 One could potentially consider a more elaborate implementation where a central
 management entity explicitly deals with `Fabric` life cycle, association of sites to
 a fabric, etc. The role of this central management component in ClusterLink is currently
 delegated to users who are responsible for coordinating the transfer to certificates
 between sites, out of band.

## Initializing a new fabric

The following assume that you have access to the `clusterlink` CLI and one or more
 sites (i.e., clusters) where you'll deploy ClusterLink. The CLI can be downloaded
 from the ClusterLink [releases page on GitHub](TBD).

To create a new Fabric certificate authority (CA), execute the following CLI command:

```sh
clusterlink create fabric --name <fabric_name>
```

This command will create the CA files `<fabric_name>.cert` and `<fabric_name>.key` in the
 current directory. While you will need access to these files to create the sites` gateway
 certificates later, the private key file should be protected and not shared with others.

## Related tasks

Once a `Fabric` has been created and initialized, you can proceed with configuring
 [sites](./sites.md). For a complete end to end use case, refer to
 [iperf toturial](TBD).
