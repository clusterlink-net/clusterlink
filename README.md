# ClusterLink Project

## Disclaimers and Warnings

This is an incomplete work in progress, provided in the interest of sharing experience
 and gathering feedback.
 The code is pre-alpha quality right now. This means that it shouldn't be used in
 production at all.

## What Is ClusterLink?

The ClusterLink project simplifies the connection between application services that are
 located in different domains, networks, and cloud infrastructures.

For more details, see the document: [ClusterLink extended abstract](docs/ClusteLink.pdf).

ClusterLink deploys a gateway into each location, facilitating the configuration and
 access to multi-cloud services.

The ClusterLink gateway contains the following components:

1. ```Control Plane``` is responsible for maintaining the internal state of the gateway,
 for all the communications with the remote peer gateways by means of the ClusterLink CP
 Protocol (REST APIs), and for commanding the local DP to forward user traffic according
 to policies.
 Part of the control plane is the policy engine that can also apply network policies
 (ACL, load-balancing, etc.)
1. ```Data Plane``` responds to user connection requests, both local and remote,
 initiates policy resolution in the CP, and maintains the established connections.
 ClusterLink DP relies upon standard protocols and avoids redundant encapsulations,
 presenting itself as a K8s service inside the cluster and as a regular HTTP endpoint
 from outside the cluster, requiring only a single open port (HTTP/443) and leveraging
 HTTP endpoints for connection multiplexing.
1. ```gwctl``` is CLI implementation that uses REST APIs to send control messages to the
 ClusterLink Gateway.

![alt text](./docs/clusterlink.png)

The ClusterLink APIs use the following entities for configuring cross cluster communication:

- Peer. Represent remote ClusterLink gateways and contain the metadata necessary for
 creating protected connections to these remote peers.
- Exported service. Represent application services hosted in the local cluster and
 exposed to remote ClusterLink gateways as Imported Service entities in those peers.
- Imported service. Represent remote application services that the gateway makes
 available locally to clients inside its cluster.
- Policy. Represent communication rules that must be enforced for all cross-cluster
 communications at each ClusterLink gateway.

## Getting Started

### Building ClusteLink

<!-- We have a [tutorial](TODO missing link) that walks you through setting up your developer
 environment, making a change and testing it.-->

Here are the key steps for setting up your developer environment, making a change and testing it:

1. Install Go version 1.20 or higher.
1. Clone our repository with `git clone git@github.com:clusterlink-net/clusterlink.git`.
1. Run `make test-prereqs` and manually install any missing required development tools.
1. Run `make build` to ensure the code builds fine. This will pull in all needed
 dependencies.
1. If you are planning on contributing back to the project, please see our
 [contribution guide](CONTRIBUTING.md).

### How to setup and run ClusterLink

ClusterLink can be set up and run on different environments: local environment (Kind),
 Bare-metal environment, or cloud environment. For more details, refer to the [Installation Guide for ClusterLink](docs/installation.md).

#### Run ClusterLink in local environment (Kind)

ClusterLink can run in any K8s environment, such as Kind.
 To run the ClusterLink in a Kind environment, follow one of the examples:

1. Performance example - Run iPerf3 test between iPerf3 client and server using ClusterLink
 components. This example is used for performance measuring. Instructions can be found
 [Here](demos/iperf3/kind/README.md).
1. Application example - Run the BookInfo application in different clusters using ClusterLink
 components. This example demonstrates communication distributed applications (in different
 clusters) with different policies.Instructions can be found [Here](demos/bookinfo/kind/README.md).

#### Run ClusterLink in Bare-metal environment with 2 hosts

TBD

#### Run ClusterLink in cloud environment

TBD

## Contributing

Our project welcomes contributions from any member of our community. To get
 started contributing, please see our [Contributor Guide](CONTRIBUTING.md).

## Scope

### In Scope

ClusterLink is intended to connect services and applications running in different clusters.
 As such, the project will implement or has implemented:

- Remote Service sharing
- Extending private Cloud service endpoints to remote sites
- Centralized management (future)

### Out of Scope

ClusterLink will be used in a cloud native environment with other
 tools. The following specific functionality will therefore not be incorporated:

- Certificate management: ClusterLink uses certificates and trust bundles provided to
 it. It does not manage certificate lifetimes, rotation, etc. - these are delegated to external tools.
- Enabling IP level connectivity between sites. ClusterLink uses existing network paths.
- Pod to Pod communications. ClusterLink works at the level of `Service`s (but you could create a Service per Pod
 if that makes sense in your use cases...)

## Communications

<!-- Fill in the communications channels you actually use.  These should all be public
 channels anyone can join, and there should be several ways that users and contributors
 can reach project maintainers. If you have recurring/regular meetings, list those or a
 link to a publicly-readable calendar so that prospective contributors know when and 
 where to engage with you. -->

- [User Mailing List](https://groups.google.com/g/clusterlink-users)
- [Developer Mailing List](https://groups.google.com/g/clusterlink-dev)
<!--
- Slack Channel:
- Public Meeting Schedule and Links:
- Social Media:
- Other Channel(s), If Any:
 -->

<!--
## Resources

[TODO: Add links to other helpful information (roadmap, docs, website, etc.)]
-->

## License

This project is licensed under [Apache License, v2.0](LICENSE).
 Code contributions require [Developer Certificate of Originality](CONTRIBUTING.md#developer-certificate-of-origin).

## Code of Conduct

We follow the [CNCF Code of Conduct](CODE_OF_CONDUCT.md).
