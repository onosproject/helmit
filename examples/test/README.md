# Atomix Test Example

To run the test:

```bash
helmet test ./cmd/helmet-examples --context ../onos-helm-charts
```

The test example includes tests for [Atomix](https://atomix.io) map operations against a Raft database.

To run the tests, use the `./cmd/helmet-examples` command, passing the `examples/charts` directory as the
test context:

```bash
helmet test ./cmd/helmet-examples \
    --suite atomix \
    --context examples/charts
```

To run a specific test, specify the benchmark name with the `--test` flag:

```bash
helmet bench ./cmd/helmet-examples \
    --suite atomix \
    --test TestMap \
    --context examples/charts
```

To change the size of the Raft database, set the `atomix-raft` chart values:

```bash
helmet test ./cmd/helmet-examples \
    --suite atomix \
    --context examples/charts \
    --set atomix-raft.clusters=3 \
    --set atomix-raft.partitions=9 \
    --set atomix-raft.backend.replicas=3
```
