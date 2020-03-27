# Atomix Simulation Example

The simulation example includes simulators for [Atomix](https://atomix.io) map put, get, and remove operations
against a Raft database.

To run the simulation, use the `./cmd/helmit-examples` command, passing the `examples/charts` directory as the
simulation context:

```bash
helmit sim ./examples/simulation/cmd \
    --suite atomix \
    --context examples/charts \
    --duration 5m
```

To change the size of the Raft database, set the `atomix-raft` chart values:

```bash
helmit sim ./examples/simulation/cmd \
    --suite atomix \
    --context examples/charts \
    --duration 5m \
    --set atomix-raft.clusters=3 \
    --set atomix-raft.partitions=9 \
    --set atomix-raft.backend.replicas=3
```
