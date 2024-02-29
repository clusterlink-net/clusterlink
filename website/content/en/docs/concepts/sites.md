---
title: Sites
description: Defining ClusterLink Sites as part of Fabric
weight: 34
---

A `Site` represents a location, such as a Kubernetes cluster, participating in a
 [Fabric]({{< ref "fabric" >}}). Each site may host one or more [Services]({{< ref "services" >}})
 it wishes to share with other sites. A site is managed by a site administrator,
 which is responsible for running the ClusterLink control and data planes. The
 administrator will typically deploy the ClusterLink components by configuring
 the [deployment CRD]({{< ref "getting-started#deploy-crd-instance" >}}). They may also wish to provide
 (often) coarse-grained access policies in accordance with high level corporate
 policies (e.g., "production sites should only communicate with other production sites").

Once a `Site` has been added to a `Fabric`, it can communicate with any other `Site`
 belonging to it. All configuration relating to service sharing (e.g., the exporting
 and importing of Services, and the setting of fine grained application policies) can be
 done with lowered privileges (e.g., by users, such as application owners).

## Initializing a new Site

### Prerequisites

The following assume that you have access to the `clusterlink` CLI and one or more
 sites (i.e., clusters) where you'll deploy ClusterLink. The CLI can be downloaded
 from the ClusterLink [releases page on GitHub](https://github.com/clusterlink-net/clusterlink/releases/latest).
 It also assumes that you have access to the [previously created]({{< ref "fabric#create-a-new-fabric-ca" >}})
 Fabric CA files.

### Create a new Site certificate

Creating a new Site is a **Fabric** administrator level operation and should be appropriately protected.

To create a new Site certificate belonging to a fabric, confirm that the Fabric CA files
 are available in the current working directory, and then execute the following CLI command:

> Note: The Fabric CA files (certificate and private key) are expected in the current
> working directory (i.e., `./<fabric_name>.crt` and `./<fabric_name>.key`).

```sh
clusterlink create site --name <site_name> --fabric <fabric_name>
```

This will create the certificate and private key files (`<site_name>.cert` and
 `<site_name>.key`, respectively) for the new site. By default, the files are
 created in a subdirectory named `<site_name>` under the current working directory.
 You can override the default by setting the `--output <path>` option.

You will need the CA certificate (but **not** the CA private key) and the site certificate
 and private in the next step. They can be provided out of band (e.g., over email) to the
 site administrator.

### Deploy ClusterLink to a Site

This operation is typically done by a local *Site administrator*, typically different
 than the *Fabric administrator*. Before proceeding, ensure that the CA certificate
 (the CA private key is not needed), and the site certificate and key files which were
 created in the previous step are in the current working directory.

1. Install the ClusterLink deployment operator.

    ```sh
    clusterlink site init
    ```

    The command assumes that kubectl is set to the correct context and credentials
    and that the certificates were created in the local folder. If they were not,
    add `-f <path>` to set the correct path to the certificate files.

    This command will deploy the ClusterLink deployment CRDs using the current
    `kubectl` context. The operation requires cluster administrator privileges
    in order to install CRDs into the cluster.
    The ClusterLink operator is installed to the `clusterlink-operator` namespace
    and the CA and site certificate and key are set as Kubernetes secrets
    in the namespace. You can confirm the successful completion of the step using
    the following commands (TODO: describe output):

    ```sh
    kubectl get crds
    ```

    and

    ```sh
    kubectl get secret --namespace clusterlink-operator
    ```

1. Deploy ClusterLink CRD instance.

    After the operator is installed, you can deploy ClusterLink by applying
    the ClusterLink instance CRD.
    Refer to the [getting started guide]({{< ref "getting-started#setup" >}}) for a description
    of the CRD instance fields.

## Related tasks

Once a `Site` has been created and initialized with the ClusterLink control and data
 planes, you can proceed with configuring [services]({{< ref "services" >}})
 and [policies]({{< ref "policies" >}}).
 For a complete end to end use case, refer to [iperf toturial]({{< ref "iperf" >}}).
