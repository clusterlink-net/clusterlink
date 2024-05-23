---
title: Peer
description: Instruction for setting up peers.
draft: true
---

Add the remote peer to each cluster:

*Client cluster*:

{{< tabpane text=true >}}
{{% tab header="File" %}}

```sh
export SERVER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' server-control-plane`
curl -s $TEST_FILES/clusterlink/peer-server.yaml | envsubst | kubectl apply -f -
```

{{% /tab %}}
{{% tab header="Full CR" %}}

```sh
export SERVER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' server-control-plane`
echo "
apiVersion: clusterlink.net/v1alpha1
kind: Peer
metadata:
  name: server
  namespace: clusterlink-system
spec:
  gateways:
    - host: "${SERVER_IP}"
      port: 30443
" | kubectl apply -f -
```

{{% /tab %}}
{{< /tabpane >}}

*Server cluster*:

{{< tabpane text=true >}}
{{% tab header="File" %}}

```sh
export CLIENT_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' client-control-plane`
curl -s $TEST_FILES/clusterlink/peer-client.yaml | envsubst | kubectl apply -f -
```

{{% /tab %}}
{{% tab header="Full CR" %}}

```sh
export CLIENT_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' client-control-plane`
echo "
apiVersion: clusterlink.net/v1alpha1
kind: Peer
metadata:
  name: client
  namespace: clusterlink-system
spec:
  gateways:
    - host: "${CLIENT_IP}"
      port: 30443
" | kubectl apply -f -
```

{{% /tab %}}
{{< /tabpane >}}

