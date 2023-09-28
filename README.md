# ClusterLink Project

## Disclaimers and Warnings

This is an incomplete work in progress, provided in the interest of sharing experience and gathering feedback.
The code is pre-alpha quality right now. This means that it shouldn't be used in production at all.

Before settling on the `ClusterLink` name. the code base was previously developed as part of a
"multi-border Gateway" concept. You may still find various places in code and documentation referring to
the old name (directly or using the `MBG` acronym). We're in the process of changing all such references.

## What Is ClusterLink?

The ClusterLink project simplifies the connection between application services that are located in different domains, networks, and cloud infrastructures.

For more details, see the document: [ClusterLink extended abstract](docs/ClusteLink.pdf).

ClusterLink deploys a gateway into each location, facilitating the configuration and access to multi-cloud services.
The ClusterLink gateway contains the following components:

1) ```Control Plane``` is responsible for maintaining the internal state of the gateway, for all the communications with the remote peer gateways by means of the ClusterLink CP Protocol (REST APIs), and for commanding the local DP to forward user traffic according to policies.
   Part of the control plane is the policy engine that can also apply network policies (ACL, load-balancing, etc.)

2) ```Data Plane``` responds to user connection requests, both local and remote, initiates policy resolution in the CP, and maintains the established connections. ClusterLink DP relies upon standard protocols and avoids redundant encapsulations, presenting itself as a K8s service inside the cluster and as a regular HTTP endpoint from outside the cluster, requiring only a single open port (HTTP/443) and leveraging HTTP endpoints for connection multiplexing.
3) ```gwctl``` is CLI implementation that uses REST APIs to send control messages to the ClusterLink Gateway.

![alt text](./docs/clusterlink.png)

The ClusterLink APIs use the following entities for configuring cross cluster communication:

* Peer. Represent remote ClusterLink gateways and contain the metadata necessary for creating protected connections to these remote peers.
* Exported service. Represent application services hosted in the local cluster and exposed to remote ClusterLink gateways as Imported Service entities in those peers.
* Imported service. Represent remote application services that the gateway makes available locally to clients inside its cluster.
* Policy. Represent communication rules that must be enforced for all cross-cluster communications at each ClusterLink gateway.

## How to setup and run ClusterLink

ClusterLink can be set up and run on different environments: local environment (Kind), Bare-metal environment, or cloud environment.

### Run ClusterLink in local environment (Kind)

ClusterLink can run in any K8s environment, such as Kind.
To run the ClusterLink in a Kind environment, follow one of the examples:

1) Performance example - Run iPerf3 test between iPerf3 client and server using ClusterLink components. This example is used for performance measuring. Instructions can be found [Here](demos/iperf3/kind/README.md).
1) Application example - Run the BookInfo application in different clusters using ClusterLink components. This example demonstrates communication distributed applications (in different clusters) with different policies.Instructions can be found [Here](demos/bookinfo/kind/README.md).

### Run ClusterLink in Bare-metal environment with 2 hosts

TBD

### Run ClusterLink in cloud environment

TBD
