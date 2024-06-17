
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
