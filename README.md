# Helmit

### Safety first!

[![Build Status](https://travis-ci.com/onosproject/helmit.svg?branch=master)](https://travis-ci.org/onosproject/helmit)
[![Go Report Card](https://goreportcard.com/badge/github.com/onosproject/helmit)](https://goreportcard.com/report/github.com/onosproject/helmit)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/gojp/goreportcard/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/onosproject/helmit?status.svg)](https://godoc.org/github.com/onosproject/helmit)

Helmit is a [Golang] framework and tool for end-to-end testing of [Kubernetes] applications.
The Helmit Go API and the `helmit` command line tool work together to manage the deployment, testing, benchmarking,
and verification of [Helm]-based applications running in Kubernetes.

Helmit can be used to:

* Verify [Helm] charts and the resources they construct
* Run end-to-end tests to verify a service/API
* Run end-to-end benchmarks for Kubernetes applications
* Scale benchmarks across multi-node Kubernetes clusters
* Run randomized client simulations for Kubernetes applications (e.g. for formal verification)

## User Guide

* [The `helmit` Tool](./docs/cli.md)
* [Testing](./docs/testing.md)
* [Benchmarking](./docs/benchmarking.md)
* [Randomized Simulations](./docs/simulation.md)
* [Kubernetes/Helm API](./docs/api.md)

## Examples

* [Basic test example](https://github.com/onosproject/helmit/tree/master/examples/test)
* [Basic benchmark example](https://github.com/onosproject/helmit/tree/master/examples/benchmark)
* [Basic simulation example](https://github.com/onosproject/helmit/tree/master/examples/simulation)
* [Working with Helm charts in tests](https://github.com/onosproject/helmit/tree/master/examples/charts)

## Acknowledgements

Helmit is a project of the [Open Networking Foundation][ONF].

![ONF](https://3vf60mmveq1g8vzn48q2o71a-wpengine.netdna-ssl.com/wp-content/uploads/2017/06/onf-logo.jpg)

[Golang]: https://golang.org/
[Helm]: https://helm.sh
[Kubernetes]: https://kubernetes.io
[ONF]: https://opennetworking.org
