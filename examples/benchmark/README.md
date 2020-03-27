# Atomix Benchmark Example

The benchmark example includes benchmarks for [Atomix](https://atomix.io) map put, get, and remove operations
against a Raft database.

To run the benchmarks, use the `./cmd/helmit-examples` command, passing the `examples/charts` directory as the
benchmark context:

```bash
helmit bench ./examples/benchmark/cmd \
    --suite atomix \
    --context examples/charts \
    --duration 5m
```

To run a specific benchmark, specify the benchmark name with the `--benchmark` flag:

```bash
helmit bench ./examples/benchmark/cmd \
    --suite atomix \
    --benchmark BenchmarkMapPut \
    --context examples/charts \
    --duration 5m
```

To change the size of the Raft database, set the `atomix-raft` chart values:

```bash
helmit bench ./examples/benchmark/cmd \
    --suite atomix \
    --benchmark BenchmarkMapPut \
    --context examples/charts \
    --duration 5m \
    --set atomix-raft.clusters=3 \
    --set atomix-raft.partitions=9 \
    --set atomix-raft.backend.replicas=3
```
