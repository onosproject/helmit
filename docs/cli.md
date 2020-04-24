## Command-Line Tools

The `helmit` command-line tool is used to run tests, benchmarks, and simulations inside a Kubernetes cluster. To
install the `helmit` CLI, use `go get` with Go modules enabled:

```bash
GO111MODULE=on go get github.com/onosproject/helmit/cmd/helmit
```

To use the Helmit CLI, you must have [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/) installed and
configured. Helmit will use the Kubernetes configuration to connect to the cluster to deploy and run tests.

The Helmit CLI consists of only three commands:

* `helmit test` - Runs a [test](#testing) command
* `helmit bench` - Runs a [benchmark](#benchmarking) command
* `helmit sim` - Runs a [simulation](#simulation) command

Each command deploys and runs pods which can deploy Helm charts from within the Kubernetes cluster using the
[Helm API](#helm-api). Each Helmit command supports configuring Helm values in the same way the `helm` command
itself does.

The `helmit` sub-commands support an optional context within which to run tests. When the `--context` flag is set,
the specified context directory will be copied to the Helmit pod running inside Kubernetes and set as the current 
working directory during runs:

```bash
helmit test ./cmd/tests --context ./deploy/charts
```

This allows suites to reference charts by path from within Helmit containers deployed inside Kubernetes:

```go
helm.Chart("./atomix-controller").
	Release("atomix-controller-1").
	Install(true)
```

As with Helm, the `helmit` commands also support values files and flags:

```bash
helmit test ./cmd/tests -f atomix-controller-1=atomix-values.yaml --set atomix-controller-1.replicas=2
```

Because suites may install multiple Helm releases, values files and flags must be prefixed by the *release* name. 
For example, `-f my-release=values.yaml` will add a values file to the release named `my-release`, and
`--set my-release.replicas=3` will set the `replicas` value for the release named `my-release`.

[Golang]: https://golang.org/
[Helm]: https://helm.sh
[Kubernetes]: https://kubernetes.io
[ONOS]: https://onosproject.org
[Atomix]: https://atomix.io
