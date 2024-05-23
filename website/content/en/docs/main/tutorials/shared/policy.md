---
title: Policy
description: Instruction for setting up policies.
draft: true
---
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

For more details regarding policy configuration, see [policies](../../concepts/policies) documentation.
