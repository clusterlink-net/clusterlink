---
title: Fabric
description: Defining a ClusterLink fabric
weight: 10
---

The concept of a `Fabric` encapsulates a set of cooperating [peers][concept-peer].
 All peers in a fabric can communicate and may share [services][concept-service]
 between them, with access governed by [policies][concept-policy].
 The `Fabric` acts as a root of trust for peer to peer communications (i.e.,
 it functions as the certificate authority enabling mutual authentication between
 peers).

Currently, the concept of a `Fabric` is just that - a concept. It is not represented
 or backed by any managed resource in a ClusterLink deployment. Once a `Fabric` is created,
 its only relevance is in providing a certificate for use by each peer's gateways.
 One could potentially consider a more elaborate implementation where a central
 management entity explicitly deals with `Fabric` life cycle, association of peers to
 a fabric, etc. The role of this central management component in ClusterLink is currently
 delegated to users who are responsible for coordinating the transfer of certificates
 between peers, out of band.

## Initializing a new fabric

### Prerequisites

The following assume that you have access to the `clusterlink` CLI and one or more
 peers (i.e., clusters) where you'll deploy ClusterLink. The CLI can be downloaded
 from the ClusterLink [releases page on GitHub](https://github.com/clusterlink-net/clusterlink/releases/latest).

### Create a new fabric CA

To create a new fabric certificate authority (CA), execute the following CLI command:

```sh
clusterlink create fabric --name <fabric_name>
```

This command will create the CA files `cert.pem` and `key.pem` in a directory named <fabric_name>.
The `--name` option is optional, and by default, "default_fabric" will be used.
While you will need access to these files to create the peers` gateway certificates later,
the private key file should be protected and not shared with others.

## Related tasks

Once a `Fabric` has been created and initialized, you can proceed with configuring
 [peers][concept-peer]. For a complete, end to end, use case please refer to the
 [iperf tutorial][tutorial-iperf].

[concept-peer]: {{< relref "peers" >}}
[concept-service]: {{< relref "services" >}}
[concept-policy]: {{< relref "policies" >}}
[tutorial-iperf]: {{< relref "../tutorials/iperf" >}}
