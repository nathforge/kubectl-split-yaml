# kubectl-split-yaml

A `kubectl` plugin to split Kubernetes YAML output into one file per resource.

## Example

```shell
$ kubectl get all -o yaml | kubectl split-yaml -p resources
resources/v1--Pod/default--nginx-86c57db685-4vnjc.yaml
resources/v1--Service/default--nginx.yaml
resources/apps_v1--Deployment/default--nginx.yaml
```

## Usage

`kubectl split-yaml [flags]`, where `[flags]` can contain:
 * `-f`: input file. If not given, or set to `-`, standard input is used.
 * `-p`: output directory. Defaults to `.`.
 * `-t`: filename template: Defaults to `{{.apiVersion}}--{{.kind}}/{{.namespace}}--{{.name}}.yaml`

## Download

Binaries are available from the [releases page](https://github.com/nathforge/kubectl-split-yaml/releases).

## Recommended projects

 * [ketall](https://github.com/corneliusweig/ketall): like `kubectl get all`, but gets *everything*.
 * [kubectl-neat](https://github.com/itaysk/kubectl-neat): removes clutter from Kubernetes resources; metadata, status, etc.

## License

Apache 2.0. See [LICENSE](LICENSE).
