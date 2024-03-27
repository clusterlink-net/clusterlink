---
title: Fabric
description: Defining a ClusterLink fabric
weight: 32
---

The concept of a `Fabric` encapsulates a set of cooperating [peers]({{< ref "peers" >}}/).
 All peers in a fabric can communicate and may share [services]({{< ref "services" >}})
 between them, with access governed by [policies]({{< ref "policies" >}}).
 The `Fabric` acts as a root of trust for peer to peer communications (i.e.,
 it functions as the certificate authority enabling mutual authentication between
 peers).

Currently, the concept of a `Fabric` is just that - a concept. It is not represented
 or backed by any managed resource in a ClusterLink deployment. Once a `Fabric` is created,
 its only relevance is in providing a certificate for use by each peer's gateways.
 One could potentially consider a more elaborate implementation where a central
 management entity explicitly deals with `Fabric` life cycle, association of peers to
 a fabric, etc. The role of this central management component in ClusterLink is currently
 delegated to users who are responsible for coordinating the transfer to certificates
 between peers, out of band.

## Initializing a new fabric

### Prerequisites

The following assume that you have access to the `clusterlink` CLI and one or more
 peers (i.e., clusters) where you'll deploy ClusterLink. The CLI can be downloaded
 from the ClusterLink [releases page on GitHub](https://github.com/clusterlink-net/clusterlink/releases/latest).

### Create a new Fabric CA

To create a new Fabric certificate authority (CA), execute the following CLI command:

```sh
clusterlink create fabric --name <fabric_name>
```

This command will create the CA files `<fabric_name>.cert` and `<fabric_name>.key` in the
 current directory. While you will need access to these files to create the peers` gateway
 certificates later, the private key file should be protected and not shared with others.

## Related tasks

Once a `Fabric` has been created and initialized, you can proceed with configuring
 [peers]({{< ref "peers" >}}). For a complete, end to end, use case please refer to the
 [iperf toturial]({{< ref "iperf" >}}).
