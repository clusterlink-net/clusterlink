---
title: Deployment Operator
description: Defining ClusterLink  deployment operator
weight: 50
---

The ClusterLink deployment operator allows easy deployment of the ClusterLink project to the K8s cluster. The preferred deployment approach involves utilizing the ClusterLink CLI, which automatically generates both the ClusterLink operator and ClusterLink components. However, it's important to note that ClusterLink deployment necessitates certificate peers for proper functioning. Detailed instructions for creating these certificate peers can be found in the [getting started](https://clusterlink.net/docs/getting-started/users/#Setup).

## The common use-case

The common use case for deploying ClusterLink on a K8s cluster (i.e., EKS, GKE, IKS, etc.) is using the CLI command:

```sh
clusterlink deploy peer --autostart --name <peer_name> --fabric <fabric_name>
```

The command assumes that `kubectl` is set to the correct peer (K8s cluster)
and that the certificates exist on the same working directory.
If they were not, use the flag `--path <path>` for pointing to the certificate directory.
The command deploys the ClusterLink operator in the `clusterlink-operator` namespace and converts the peer certificates to secrets in the `clusterlink-system` namespace, where ClusterLink components will be installed. By default, these components are deployed in the `clusterlink-system` namespace. The `--autostart` option deploys the ClusterLink components in the `clusterlink-system` namespace and enables ClusterLink in the cluster. Additionally, a `LoadBalancer` service is created to allow cross-cluster connectivity using ClusterLink.

## Deployment for Kind environment

To deploy ClusterLink in a local environment like Kind, you can use the following command:

```sh
clusterlink deploy peer --autostart --name <peer_name> --fabric <fabric_name> --ingress=NodePort --ingress-port=30443
```

The Kind environment doesn't allocate an external IP to the `LoadBalancer` service by default. In this case, we will use a `NodePort` service to establish multi-cluster connectivity using ClusterLink.

## Deployment of specific version

To deploy a specific ClusterLink version use the `tag` flag:

```sh
clusterlink deploy peer --autostart --name <peer_name> --fabric <fabric_name> --tag <version_tag>
```

## Deployment using ClusterLink YAML

The deployment process can be split into two steps:

1. Deploy only ClusterLink operator:

    ```sh
    clusterlink deploy peer ---name <peer_name> --fabric <fabric_name>
    ```

    This command will deploy only the ClusterLink operator and the certificate's secrets.

1. {{< anchor deploy-cr-instance >}} Deploy a ClusterLink K8s custom resource object:

    ```yaml
    kubectl apply -f - <<EOF
    apiVersion: clusterlink.net/v1alpha1
    kind: ClusterLink
    metadata:
        namespace: clusterlink-operator
        name: <peer_name>
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

## Additional deployment configurations

The `deploy peer` command has the following flags:

- **namespace:** This field determines the namespace where the ClusterLink components are deployed. By default, it uses `clusterlink-system`, which is created by the `deploy peer` command. If a different namespace is desired, it must already exist.
- **dataplane:**  This field determines the type of ClusterLink dataplane, with supported values `go` or `envoy`. By default, it uses `envoy`.
- **dataplane-replicas:** This field determines the number of ClusterLink dataplane replicas. By default, it uses 1.
- **ingress:** This field determines the type of ingress service to expose ClusterLink deployment, with supported values: `LoadBalancer`, `NodePort`, or `None`. By default, it uses `LoadBalancer`.
- **ingress-port:** This field determines the port number of the external service. By default, it uses port `443` for all types, except for `NodePort`, where the port number will be allocated by Kubernetes.
- **log-level:** This field determines the severity log level for all the components (controlplane and dataplane). By default, it uses `info` log level.
- **container-registry:** This field determines the container registry to pull the project images. By default, it uses `ghcr.io/clusterlink-net`.
- **tag:** This field determines the version of project images to pull. By default, it uses the `latest` version.

All the flags are mapped to corresponding YAML fields.
