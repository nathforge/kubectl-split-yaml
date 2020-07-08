# kubectl-split-yaml

A `kubectl` plugin to split Kubernetes YAML output into one file per
resource.

## Example

```shell
$ kubectl get all -o yaml | kubectl split-yaml .
v1--Pod/default--nginx-86c57db685-4vnjc.yaml
v1--Service/default--nginx.yaml
apps_v1--Deployment/default--nginx.yaml
apps_v1--ReplicaSet/default--nginx-86c57db685.yaml
```

## License

Apache 2.0. See [LICENSE](LICENSE).
