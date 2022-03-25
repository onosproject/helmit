<!--
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
-->

## Testing

Helmit supports testing of [Kubernetes] resources and [Helm] charts using a custom test framework and
command line tool. To test a Kubernetes application, simply [write a Golang test suite](#writing-tests)
and then run the suite using the `helmit test` tool:

```bash
helmit test ./cmd/tests
```

### Writing Tests

Helmit tests are written as suites. When tests are run, each test suite will be deployed and run in its own namespace.
Test suite functions are executed serially.

```go
import "github.com/onosproject/helmit/pkg/test"

type AtomixTestSuite struct {
	test.Suite
}
```

The `SetupTestSuite` interface can be implemented to set up the test namespace prior to running tests:

```go
func (s *AtomixTestSuite) SetupTestSuite() error {
	err := helm.Chart("atomix-controller").
        Release("atomix-controller").
        Set("scope", "Namespace").
        Install(true)
    if err != nil {
        return err
    }

    err = helm.Chart("atomix-database").
        Release("atomix-raft").
        Set("clusters", 3).
        Set("partitions", 10).
        Set("backend.replicas", 3).
        Set("backend.image", "atomix/raft-replica:latest").
        Install(true)
    if err != nil {
        return err
    }
    return nil
}
```

Tests are receivers on the test suite that follow the pattern `Test*`. The standard Golang `testing` library is
used, so all your favorite assertion libraries can be used as well:

```go
import "testing"

func (s *AtomixTestSuite) TestMap(t *testing.T) {
    address, err := s.getController()
    assert.NoError(t, err)

    client, err := atomix.New(address)
    assert.NoError(t, err)

    database, err := client.GetDatabase(context.Background(), "atomix-raft")
    assert.NoError(t, err)

    m, err := database.GetMap(context.Background(), "TestMap")
    assert.NoError(t, err)
}
```

Helmit also supports `TearDownTestSuite` and `TearDownTest` functions for tearing down test suites and tests 
respectively:

```go
func (s *AtomixTestSuite) TearDownTest() error {
	return helm.Chart("atomix-database").
		Release("atomix-database").
		Uninstall()
}
```

### Registering Test Suites

In order to run tests, a main must be provided that registers and names test suites.

```go
import "github.com/onosproject/helmit/pkg/registry"

func init() {
	registry.RegisterTestSuite("atomix", &tests.AtomixTestSuite{})
}
```

Once the tests have been registered, the main should call `test.Main()` to run the tests:

```go

import (
	"github.com/onosproject/helmit/pkg/registry"
	"github.com/onosproject/helmit/pkg/test"
	tests "github.com/onosproject/helmit/test"
)

func main() {
	registry.RegisterTestSuite("atomix", &tests.AtomixTestSuite{})
	test.Main()
}
```

### Running Tests

Once a test suite has been written and registered, running the tests on Kubernetes is simply a matter of running
the `helmit test` command and pointing to the test main:

```bash
helmit test ./cmd/tests
```

When `helmit test` is run with no additional arguments, the test coordinator will run all registered test suites 
in parallel and each within its own namespace. To run a specific test suite, use the `--suite` flag:

```bash
helmit test ./cmd/tests --suite my-tests
```

The `helmit test` command also supports configuring tested Helm charts from the command-line. See the 
[command-line tools](#command-line-tools) documentation for more info.
