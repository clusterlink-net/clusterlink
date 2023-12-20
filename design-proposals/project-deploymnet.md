# Design Proposal: Project Deployment with Kubernetes Operator

**Author**: @kfirtoledo

**Begin Design Discussion**: 2023-11-30

**Status:** draft

## Summary/Abstract

The ClusterLink project allows the interconnection of various clusters within the same fabric. For each fabric, there is a fabric administrator responsible for determining which clusters can join the fabric, and a site administrator responsible for deploying ClusterLink to the Kubernetes (K8s) cluster.

This design proposal outlines the deployment process for the ClusterLink project on a Kubernetes cluster by the site administrator. It relies on a dedicated K8s operator to make the deployment more user-friendly, simplified, and native to K8s users.

## Background

### Motivation and problem space

This design proposal aims to simplify and enhance the deployment of the ClusterLink project.
By leveraging K8s native mechanisms, such as operators and CRDs, the objective is to establish a user-friendly deployment process.
Currently, ClusterLink deployment involves using the cl-adm CLI, which generates a large YAML file that contains the control plane and dataplane deployments, services, and RBAC configurations.
In the new design, this process will be managed by the ClusterLink operator, which has the following advantages:

* It consumes a simpler and more concise YAML configuration.
* The use of CRD and the operator aids in automating deployment changes over time.
* The proposal differentiates between fabric and site administrators and provides a more secure deployment model via k8s RBAC.

### Impact and desired outcome

Implementing this proposal is expected to improve and simplify the deployment process for the ClusterLink users.
Aligning the deployment process with K8s standards will particularly benefit use cases involving central control management.

### Prior discussion and links

Not applicable.

## User/User Story

As a site administrator, I would like to join my cluster to an existing fabric. The fabric administrator should send me the certificates required to join the fabric: the fabric CA certificate and the cluster certificates (private and public keys). Using these certificates and the ClusterLink operator, I can deploy the ClusterLink system in my cluster. Now, any user in my cluster can connect to other remote services in the fabric, even if they are located on different locations, or different cloud providers.

## Goals

This design document should:

* Define the steps for deploying a ClusterLink gateway to a K8s cluster.
* Define the task functions of the ClusterLink deployment operator for setting up the ClusterLink components in the cluster.
* Define the CRD (Custom Resource Definition) for the ClusterLink deployment operator.

## Non-Goals

The following aspects are explicitly excluded and are out of scope in the current design:

* This document focuses only on the deployment of ClusterLink to the K8s cluster.
The deployment to another environment as VMs is out of scope.
* Security considerations related to the creation of certificates for the fabric or each site by the fabric administrator are also beyond the scope of this document.

## Proposal

The proposal describes how the ClusterLink project can be deployed on any K8s Cluster. In this proposal, we distinguish between two entities: first, the fabric administrator, who manages the fabric and is responsible for determining which clusters can join the fabric, and second, the site administrator, who manages the k8s Cluster and is responsible for deploying the ClusterLink into the cluster.

Before deploying the ClusterLink, a few prerequisite steps should be completed:

1. The fabric administrator should create the site certificates (public and private) and transfer them, along with the public CA certificate, to the site administrator.
1. The site administrator should deploy the certificates as k8s secrets to the cluster.
1. The site administrator should deploy the ClusterLink operator to the cluster.

After completing all the prerequisites, the site administrator can deploy the ClusterLink Custom Resource Definition (CRD) to the operator. The operator will then create the ClusterLink components, including:

* A ClusterLink dedicated namespace, which by default will be cl-operator-ns.
* Deployment of ClusterLink controlplane components, including controlplane-pod, controlplane-service, and RBAC roles.
* Deployment of ClusterLink dataplane components, including dataplane pods (single or multiple) and the dataplane-service.
* Deployment of ClusterLink ingress, providing an external access point using load-balancer/node-port/gateway-API service."

Overall, the ClusterLink deployment stages are:

<img src="deployment.png" width="800" height="400" alt="ClusterLink Deployment Stages"/>

The ClusterLink operator will have privileged permissions, enabling it to create ClusterLink components within the dedicated namespace.
The ClusterLink components will be created in `clusterlink-system-ns`, and the control-plane will have privileged permissions to execute operations on the ClusterLink CRDs, such as peer, export, import, policy, etc.
Additionally, the operator will create and update components in response to the `clusterlink-system.yaml` create or update actions. Once created or updated, the operator will not actively monitor the state of each component, assuming that any changes will be made only by privileged users.
Furthermore, the operator will delete all components and the namespace of ClusterLink when the `clusterlink-system.yaml` file is deleted.

The deployment file for the operator and example of `clusterlink-system.yaml` will be included in every project release.

The ClusterLink CLI will be utilized to automate the deployment process.
For instance, the following commands can assist the fabric administrator in creating the site certificates:

    clusterlink create fabric --name <fabric_name>
    clusterlink create site --name <site_name> --fabric <fabric_name> 

For the site administrator, automation of deploying the certificates as a secret to the cluster, deploying the ClusterLink operator, and the `clusterlink-system.yaml` can be achieved with:

    clusterlink install --site <site_name>

### ClusterLink CRD

The ClusterLink CRD includes the following fields:

* **API version:** clusterlink.net/v1alpha1
* **Kind:** clusterlink

* **Spec:**
  
    | Field name | Subfield name| Description | Default value |
    | ---- | ----- | ------ | ----|
    |dataplane | | ClusterLink dataplane component attributes||
    |  |type| Types of dataplane, supported values "go" or "envoy"|"envoy"|
    | |replicates| Number of dataplane replicas|1|
    |ingress|| ClusterLink ingress component attributes ||
    ||type| Type of service to expose ClusterLink deployment, supported values: "load-balancer", "API-gateway",    "node-port". |load-balancer|
    |logLevel| |Log level severity for all the components (controlplane and dataplane)| "info"|
    |containerRegistry| |The container registry to pull the project images when the images is not present locally | ghcr.io/clusterlink-net|
    |imageTag| |The project images version | latest|

* **Status:**

    | Field name | Description | Default value |
    | ----------- | ----------- | ------------- |
    | controlplaneStatus      | Status of the controlplane components controlled by the operator. Supported values: <br> None - if all components have not been created. <br> Ready - if all components are in a ready and running status. <br> Error - If one of the components is in an error status. <br> Update - if one of the components is in an update status. | None |
    | dataplaneStatus      | Status of the dataplane components controlled by the operator. Supported values: <br> None - if all components have not been created. <br> Ready - if all components are in a ready and running status. <br> Error - If one of the components is in an error status. <br> Update - if one of the components is in an update status. | None |
    | message     | Optional message that provides additional information about the status. | None |

Example to clusterlink CRD:
```
apiVersion: clusterlink.net/v1alpha1
kind: ClusterLink
metadata:
  namespace: clusterlink-system-ns
  name: peer1
spec:
  dataplane:
    type: "envoy"
    replicates: 1
```

## Impacts / Key Questions

* Do we need the ClusterLink CLI? The only command that seems necessary is for fabric and peer certificates.
* Need to have a security discussion focusing on how and who deploys the peer and fabric certificates.

## Risks and Mitigations

Not applicable.

### Security Considerations

For the deployment of ClusterLink, the Fabric CA certificate (public key) and the peer certificates (public and private keys) must be provided. These certificates will be deployed to the cluster using k8s-secrets and should be handled by the cluster owner. The creation of these certificates can be performed by the central management, which is outside the scope of this document.

The ClusterLink operator will have privileged permissions to deploy the ClusterLink project to the cluster. The control plane of ClusterLink will be granted privileged access to all cluster resources through RBAC (Role-Based Access Control).

In the future, we may introduce a Namespace deployment, which will have permissions limited to a single namespace.

## Implementation Details

The k8s operator will be built using the [Kubebuilder](https://kubebuilder.io/) tool, which allows easy building of k8s APIs and operators in Go.

### Testing Plan

There will be two types of tests:

1. Unit test: This test checks the operator's behavior and functionality, including the creation, updating, and deletion of ClusterLink components.
1. System-level test: This test checks the entire deployment process using CLI automation.

### Implementation Phases/History

The first phase focuses on building the k8s operator. In this step, we will create the ClusterLink operator and suitable tests. Additionally, during this step, we will continue to use the current cl-adm implementation to create peer and fabric certificates.
The `cl-adm create peer1` command will generate two files: `k8s-secret.yaml` (containing all the certificates for the control-plane and data-plane) and `clusterlink-system.yaml` (the CRD for the ClusterLink operator). The ClusterLink operator will be deployed manually by the site administrator. The site administrator deploys the `clusterlink-system.yaml` file to the ClusterLink operator, than the operator creates the ClusterLink components.

In the second step, the focus is on automating the deployment process. We will create a ClusterLink CLI. This CLI will automate certificate creation by the fabric administrator, replacing the current cl-adm CLI. Furthermore, the CLI will automate the deployment process for ClusterLink by the site manager (including secret creation, deploying the ClusterLink operator, and CRD creation).