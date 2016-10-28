# Example Usage

```
kubectl create -f replicasets/vault-example.yaml
```

```
kubectl get pods
```
```
NAME                     READY     STATUS            RESTARTS   AGE
vault-8q9rw              1/1       Running           0          22m
vault-controller-o1d04   2/2       Running           0          12m
vault-example-93w3w      0/1       PodInitializing   0          3s
```

```
kubectl logs vault-example-93w3w -c vault-init
```

```
2016/10/28 16:11:58 Starting vault-init...
2016/10/28 16:11:59 Requesting a new wrapped token from http://vault-controller:80
2016/10/28 16:11:59 Token request complete; waiting for callback...
2016/10/28 16:11:59 Successfully obtained and unwrapped the vault token, exiting...
```
