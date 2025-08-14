KinD doesn't have multiple real nodes,
  so scheduling pods based on cpu requests is useless
  and can prevent deployment altogether.

Originally written because github runners only have 2 cpus, causing a KinD CI workflow to fail.

```
./gen-certs.sh
kubectl apply -f ./k8s-resources.yaml
```
