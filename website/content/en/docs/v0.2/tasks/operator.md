---
title: Deployment Operator
description: Usage and configuration of the ClusterLink deployment operator.
weight: 50
---

The ClusterLink deployment operator allows easy deployment of ClusterLink to a K8s cluster.
The preferred deployment approach involves utilizing the ClusterLink CLI,
which automatically deploys both the ClusterLink operator and ClusterLink components.
However, it's important to note that ClusterLink deployment necessitates peer certificates for proper functioning.
Detailed instructions for creating these peer certificates can be found in the [user guide][getting-started-user-setup].

## The common use-case

The common use case for deploying ClusterLink on a cloud based K8s cluster (i.e., EKS, GKE, IKS, etc.) is using the CLI command:

```sh
clusterlink deploy peer --name <peer_name> --fabric <fabric_name>
```

The command assumes that `kubectl` is configured to access the correct peer (K8s cluster)
and that certificates files are placed in the current working directory.
If they are not, use the flag `--path <path>` to reference the directory where certificate files are stored.
The command deploys the ClusterLink operator in the `clusterlink-operator` namespace and converts
the peer certificates to secrets in the `clusterlink-system` namespace, where ClusterLink components will be installed.
By default, these components are deployed in the `clusterlink-system` namespace.
In addition, the command will create a ClusterLink instance custom resource object and deploy it to the operator.
The operator will then create the ClusterLink components in the `clusterlink-system` namespace and enable ClusterLink in the cluster.
Additionally, a `LoadBalancer` service is created to allow cross-cluster connectivity using ClusterLink.

## Deployment for Kind environment

To deploy ClusterLink in a local environment like Kind, you can use the following command:

```sh
clusterlink deploy peer --name <peer_name> --fabric <fabric_name> --ingress=NodePort --ingress-port=30443
```

The Kind environment doesn't allocate an external IP to the `LoadBalancer` service by default.
In this case, we will use a `NodePort` service to establish multi-cluster connectivity using ClusterLink.
Alternatively, you can install MetalLB to add a Load Balancer implementation to the Kind cluster. See instructions
[here](https://kind.sigs.k8s.io/docs/user/loadbalancer/).
The port flag is optional, and by default, ClusterLink will use any allocated NodePort that the Kind cluster provides.
However, it is more convenient to use a fixed setting NodePort for peer configuration, as demonstrated in the
[ClusterLink Tutorials][tutorials].

## Deployment of specific version

To deploy a specific ClusterLink image version use the `tag` flag:

```sh
clusterlink deploy peer --name <peer_name> --fabric <fabric_name> --tag <version_tag>
```

The `tag` flag will change the tag version in the ClusterLink instance custom resource object that will be deployed to the operator.

## Deployment using manually defined ClusterLink custom resource

The deployment process can be split into two steps:

1. Deploy only ClusterLink operator:

    ```sh
    clusterlink deploy peer ---name <peer_name> --fabric <fabric_name> --start operator
    ```

    The `start` flag will deploy only the ClusterLink operator and the certificate's secrets as described in the [common use case][cloud-install] above.

2. {{< anchor deploy-cr-instance >}} Deploy a ClusterLink instance custom resource object:

    ```yaml
    kubectl apply -f - <<EOF
    apiVersion: clusterlink.net/v1alpha1
    kind: Instance
    metadata:
        namespace: clusterlink-operator
        name: peer-instance
    spec:
        ingress:
            type: <ingress_type>
        dataplane:
            type: envoy
            replicas: 1
        logLevel: info
        namespace: clusterlink-system
    EOF
    ```

## Full list of the deployment configuration flags

The `deploy peer` {{< anchor commandline-flags >}} command has the following flags:

1. Flags that are mapped to the corresponding fields in the ClusterLink custom resource:

   - **namespace:** This field determines the namespace where the ClusterLink components are deployed.
    By default, it uses `clusterlink-system`, which is created by the `clusterlink deploy peer` command.
    If a different namespace is desired, that namespace must already exist.
   - **dataplane:** This field determines the type of ClusterLink dataplane, with supported values `go` or `envoy`. By default, it uses `envoy`.
   - **dataplane-replicas:** This field determines the number of ClusterLink dataplane replicas. By default, it uses 1.
   - **ingress:** This field determines the type of ingress service to expose ClusterLink deployment,
     with supported values: `LoadBalancer`, `NodePort`, or `None`. By default, it uses `LoadBalancer`.
   - **ingress-port:** This field determines the port number of the external service.
     By default, it uses port `443` for the `LoadBalancer` ingress type.
     For the `NodePort` ingress type, the port number will be allocated by Kubernetes.
     In case the user changes the default value, it is the user's responsibility to ensure the port number is valid and available for use.
   - **ingress-annotations:** This field add annotations to the ingress service.
   The flag can be repeated to add several annotations. For example: `--ingress-annotations load-balancer-type=nlb --ingress-annotations load-balancer-name=cl-nlb`.
   - **log-level:** This field determines the severity log level for all the components (controlplane and dataplane).
     By default, it uses `info` log level.
   - **container-registry:** This field determines the container registry to pull the project images.
     By default, it uses `ghcr.io/clusterlink-net`.
   - **tag:** This field determines the version of project images to pull. By default, it uses the `latest` version.

2. General deployment flags:
   - **start:** Determines which components to deploy and start in the cluster.
        `all` (defualt) starts the clusterlink operator, converts the peer certificates to secrets,
        and deploys the operator ClusterLink custom resource to create the ClusterLink components.
        `operator`, deploys only the `ClusterLink` operator and convert the peer certificates to secrets.
        `none`, doesn't deploy anything but creates ClusterLink custom resource YAML.
   - **path**: represents the path where the peer and fabric certificates are stored,
        By default is the working current working directory.

## Manual Deployment without CLI

To deploy the ClusterLink without using the CLI, follow the instructions below:

1. Download the configuration files (CRDs, operator RBACs, and deployment) from GitHub:

    ```sh
    git clone git@github.com:clusterlink-net/clusterlink.git
    ```

2. Install ClusterLink CRDs:

    ```sh
    kubectl apply --recursive -f ./clusterlink/config/crds
    ```

3. Install the ClusterLink operator:

    ```sh
    kubectl apply --recursive -f ./clusterlink/config/operator
    ```

4. Convert the peer and fabric certificates to secrets:

    ```sh
    export CERTS =<path to fabric certificates folder>
    kubectl create secret generic cl-fabric -n clusterlink-system --from-file=ca=$CERTS /cert.pem
    kubectl create secret generic cl-peer -n clusterlink-system --from-file=ca=$CERTS /peer1/cert.pem
    kubectl create secret generic cl-controlplane -n clusterlink-system --from-file=cert=$CERTS /peer1/controlplane/cert.pem --from-file=key=$CERTS /peer1/controlplane/key.pem
    kubectl create secret generic cl-dataplane -n clusterlink-system --from-file=cert=$CERTS /peer1/dataplane/cert.pem --from-file=key=$CERTS /peer1/dataplane/key.pem
    kubectl create secret generic gwctl -n clusterlink-system --from-file=cert=$CERTS /peer1/gwctl/cert.pem --from-file=key=$CERTS /peer1/gwctl/key.pem
    ```

5. Deploy a ClusterLink K8s custom resource object:

    ```yaml
    kubectl apply -f - <<EOF
    apiVersion: clusterlink.net/v1alpha1
    kind: Instance
    metadata:
        namespace: clusterlink-operator
        name: peer-instance
    EOF
    ```

[getting-started-user-setup]: {{< relref "../getting-started/users#setup" >}}
[tutorials]: {{< relref "../tutorials/" >}}
[cloud-install]: #the-common-use-case
