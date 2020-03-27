# Atomix Test Example

To run the test:

```bash
helmit test ./cmd/helmit-examples --context ../onos-helm-charts
```

The test example includes tests for [Atomix](https://atomix.io) map operations against a Raft database.

To run the tests, use the `./cmd/helmit-examples` command, passing the `examples/charts` directory as the
test context:

```bash
helmit test ./cmd/helmit-examples \
    --suite atomix \
    --context examples/charts
```

To run a specific test, specify the benchmark name with the `--test` flag:

```bash
helmit bench ./cmd/helmit-examples \
    --suite atomix \
    --test TestMap \
    --context examples/charts
```

To change the size of the Raft database, set the `atomix-raft` chart values:

```bash
helmit test ./cmd/helmit-examples \
    --suite atomix \
    --context examples/charts \
    --set atomix-raft.clusters=3 \
    --set atomix-raft.partitions=9 \
    --set atomix-raft.backend.replicas=3
```
