---
title: Nginx
description: Running basic connectivity between Nginx server and client across two clusters using ClusterLink.
---

In this tutorial, we'll establish connectivity across clusters using ClusterLink to access a remote Nginx server.
The tutorial uses two kind clusters:

1) Client cluster - runs ClusterLink along with a client.
2) Server cluster - runs ClusterLink along with a Nginx server.

## Install ClusterLink CLI

{{% include "../shared/cli-installation.md" %}}

## Initialize clusters

In this tutorial we set up a local environment using [kind][].
 You can skip this step if you already have access to existing clusters, just be sure to
 set KUBECONFIG accordingly.

To setup two kind clusters:

1. Install kind using [kind installation guide][].
1. Create a directory for all the tutorial files:

    ```sh
    mkdir nginx-tutorial
    ```

1. Open two terminals in the tutorial directory and create a kind cluster in each terminal:

    *Client cluster*:

    ```sh
    cd nginx-tutorial
    kind create cluster --name=client
    ```

    *Server cluster*:

    ```sh
    cd nginx-tutorial
    kind create cluster --name=server
    ```

   {{< notice note >}}
   kind uses the prefix `kind`, so the name of created clusters will be **kind-client** and **kind-server**.
   {{< /notice >}}

1. Setup `KUBECONFIG` on each terminal to access the cluster:

    *Client cluster*:

    ```sh
    kubectl config use-context kind-client
    cp ~/.kube/config $PWD/config-client
    export KUBECONFIG=$PWD/config-client
    ```

    *Server cluster*:

    ```sh
    kubectl config use-context kind-server
    cp ~/.kube/config $PWD/config-server
    export KUBECONFIG=$PWD/config-server
    ```

{{< notice tip >}}
You can run the tutorial in a single terminal and switch access between the clusters
using `kubectl config use-context kind-client` and `kubectl config use-context kind-server`.
{{< /notice >}}

## Deploy nginx client and server

Setup the ```TEST_FILES``` variable, and install nginx on the server cluster.

*Client cluster*:

```sh
export TEST_FILES=https://raw.githubusercontent.com/clusterlink-net/clusterlink/main/demos/nginx/testdata
```

*Server cluster*:

```sh
export TEST_FILES=https://raw.githubusercontent.com/clusterlink-net/clusterlink/main/demos/nginx/testdata
kubectl apply -f $TEST_FILES/nginx-server.yaml
```

## Deploy ClusterLink

{{% include "../shared/deploy-clusterlink.md" %}}

## Enable cross-cluster access

In this step, we enable connectivity access between the client and server.
 For each step, you have an example demonstrating how to apply the command from a
 file or providing the complete custom resource (CR) associated with the command.

Note that the provided YAML configuration files refer to environment variables
 (defined below) that should be set when running the tutorial. The values are
 replaced in the YAMLs using `envsubst` utility.

{{% expand summary="Installing `envsubst` on macOS" %}}
In case `envsubst` does not exist, you can install it with:

```sh
brew install gettext
brew link --force gettext
```

{{% /expand %}}

### Set-up peers

{{% include "../shared/peer.md" %}}

{{< notice note >}}
The `CLIENT_IP` and `SERVER_IP` refers to the node IP of the peer kind cluster, which assigns the peer YAML file.
{{< /notice >}}

### Export the Nginx server endpoint

In the server cluster, export the Nginx server service:

*Server cluster*:

{{< tabpane text=true >}}
{{% tab header="File" %}}

```sh
kubectl apply -f $TEST_FILES/clusterlink/export-nginx.yaml
```

{{% /tab %}}
{{% tab header="Full CR" %}}

```sh
echo "
apiVersion: clusterlink.net/v1alpha1
kind: Export
metadata:
  name: nginx
  namespace: default
spec:
  port:  80
" | kubectl apply -f -
```

{{% /tab %}}
{{< /tabpane >}}

### Set-up import

In the client cluster, import the Nginx service from the server cluster:

*Client cluster*:

{{< tabpane text=true >}}
{{% tab header="File" %}}

```sh
kubectl apply -f $TEST_FILES/clusterlink/import-nginx.yaml
```

{{% /tab %}}
{{% tab header="Full CR" %}}

```sh
echo "
apiVersion: clusterlink.net/v1alpha1
kind: Import
metadata:
  name: nginx
  namespace: default
spec:
  port:       80
  sources:
    - exportName:       nginx
      exportNamespace:  default
      peer:             server
" | kubectl apply -f -
```

{{% /tab %}}
{{< /tabpane >}}

### Set-up access policies

Create access policies on both clusters to allow connectivity:

*Client cluster*:

{{< tabpane text=true >}}
{{% tab header="File" %}}

```sh
kubectl apply -f $TEST_FILES/clusterlink/allow-policy.yaml
```

{{% /tab %}}
{{% tab header="Full CR" %}}

```sh
echo "
apiVersion: clusterlink.net/v1alpha1
kind: AccessPolicy
metadata:
  name: allow-policy
  namespace: default
spec:
  action: allow
  from:
    - workloadSelector: {}
  to:
    - workloadSelector: {}
" | kubectl apply -f -
```

{{% /tab %}}
{{< /tabpane >}}

*Server cluster*:

{{< tabpane text=true >}}
{{% tab header="File" %}}

```sh
kubectl apply -f $TEST_FILES/clusterlink/allow-policy.yaml
```

{{% /tab %}}
{{% tab header="Full CR" %}}

```sh
echo "
apiVersion: clusterlink.net/v1alpha1
kind: AccessPolicy
metadata:
  name: allow-policy
  namespace: default
spec:
  action: allow
  from:
    - workloadSelector: {}
  to:
    - workloadSelector: {}
" | kubectl apply -f -
```

{{% /tab %}}
{{< /tabpane >}}

For more details regarding policy configuration, see [policies][] documentation.

## Test service connectivity

Test the connectivity between the clusters with a batch job of the ```curl``` command:

*Client cluster*:

```sh
kubectl apply -f $TEST_FILES/nginx-job.yaml
```

Verify the job succeeded:

```sh
kubectl logs jobs/curl-nginx-homepage
```

{{% expand summary="Sample output" %}}

```sh
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
html { color-scheme: light dark; }
body { width: 35em; margin: 0 auto;
font-family: Tahoma, Verdana, Arial, sans-serif; }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
</html>
```

{{% /expand %}}

## Cleanup

1. Delete the kind clusters:
    *Client cluster*:

    ```sh
    kind delete cluster --name=client
    ```

    *Server cluster*:

    ```sh
    kind delete cluster --name=server
    ```

1. Remove the tutorial directory:

    ```sh
    cd .. && rm -rf nginx-tutorial
    ```

1. Unset the environment variables:
    *Client cluster*:

    ```sh
    unset KUBECONFIG TEST_FILES
    ```

    *Server cluster*:

    ```sh
    unset KUBECONFIG TEST_FILES
    ```

[kind]: https://kind.sigs.k8s.io/
[kind installation guide]: https://kind.sigs.k8s.io/docs/user/quick-start
[core concepts]: {{< relref "../../concepts/_index.md" >}}
