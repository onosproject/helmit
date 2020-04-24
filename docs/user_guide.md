## Helm API

Helmit provides a Go API for managing Helm charts within a Kubernetes cluster. Tests, benchmarks, and simulations
can use the Helmit Helm API to configure and install charts to test and query resources within releases.

The Helm API is provided by the `github.com/onosproject/helmit/pkg/helm` package:

```go
import "github.com/onosproject/helmit/pkg/helm"

chart := helm.Chart("my-chart")
release := chart.Release("my-release")
```

The Helmit API supports installation of local or remote charts. To install a local chart, use the path 
as the chart name:

```go
helm.Chart("atomix-controller").
	Release("atomix-controller").
	Install(true)
```

To install a remote chart, simply use the chart name:

```go
helm.Chart("kafka").
	Release("kafka").
	Install(true)
```

If the chart repository is not accessible from within the test container, you can optionally specify a repository
URL when creating the chart:

```go
helm.Chart("kafka", "http://storage.googleapis.com/kubernetes-charts-incubator").
	Release("kafka").
	Install(true)
```

The `Install` method installs the chart in the same was as the `helm install` command does. The boolean flags to the
`Install` method indicates whether to block until the chart's resources are ready. 

Release values can be set programmatically using the `Set` receiver:

```go
helm.Chart("kafka", "http://storage.googleapis.com/kubernetes-charts-incubator").
	Release("kafka").
	Set("replicas", 2).
	Set("zookeeper.replicaCount", 3).
	Install(true)
```

Note that values set via command line flags take precedence over programmatically configured values.

## Kubernetes Client

Tests often need to query the resources created by a Helm chart that has been installed. Helmit provides a
custom Kubernetes client designed to query Helm chart resources. The Helmit Kubernetes client looks similar
to the standard [Go client](https://github.com/kubernetes/client-go) but can limit the scope of API calls to
resources transitively owned by a Helm chart release.

To create a Kubernetes client for a release, call `NewForRelease`:

```go
// Create an atomix-controller release
release := helm.Chart("atomix-controller").Release("atomix-controller")

// Create a Kubernetes client scoped for the atomix-controller release
client := kubernetes.NewForReleaseOrDie(release)
```

The release scoped client can be used to list resources created by the release. This can be helpful for e.g.
injecting failures into the cluster during tests:

```go
// Get a list of pods created by the atomix-controller
pods, err := client.CoreV1().Pods().List()
assert.NoError(t, err)

// Get the Atomix controller pod
pod := pods[0]

// Delete the pod
err := pod.Delete()
assert.NoError(t, err)
```

Additionally, Kubernetes objects that create and own other Kubernetes resources -- like `Deployment`, `StatefulSet`, 
`Job`, etc -- provide scoped clients that can be used to query the resources they own as well:

```go
// Get the atomix-controller deployment
deps, err := client.AppsV1().Deployments().List()
assert.NoError(t, err)
assert.Len(t, deps, 1)
dep := deps[0]

// Get the pods created by the controller deployment
pods, err := dep.CoreV1().Pods().List()
assert.NoError(t, err)
assert.Len(t, pods, 1)
pod := pods[0]

// Delete the controller pod
err = pod.Delete()
assert.NoError(t, err)

// Wait a minute for the controller deployment to recover
err = dep.Wait(1 * time.Minute)
assert.NoError(t, err)

// Verify the pod was recovered
pods, err := dep.CoreV1().Pods().List()
assert.NoError(t, err)
assert.Len(t, pods, 1)
assert.NotEqual(t, pod.Name, pods[0].Name)
```

### Code Generation

Like other Kubernetes clients, the Helmit Kubernetes client is generated from a set of templates and Kubernetes
resource metadata using the `helmit-generate` tool.

```bash
go run github.com/onosproject/helmit/cmd/helmit-generate ...
```

Given a [YAML file](./build/helmit-generate/generate.yaml) defining the client's resources, the `helmit-generate` tool 
generates the scoped client code. To generate the base Helmit Kubernetes client, run `make generate`:

```bash
make generate
```

To generate a client with additional resources that are not supported by the base client, define your own
[client configuration](./build/helmit-generate/generate.yaml) and run the tool:

```go
go run github.com/onosproject/helmit/cmd/helmit-generate ./my-client.yaml ./path/to/my/package
```

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
