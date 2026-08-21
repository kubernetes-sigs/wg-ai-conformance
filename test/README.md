## Prerequisites
To run these AI Conformance tests, you must have:

- Golang: Installed on your local machine.
- Kubeconfig: A valid kubeconfig file with cluster-admin permissions for the target cluster.
- Accelerator Node Pool: The cluster must have nodes with accelerators exposed through the Kubernetes resource management framework — either a DRA driver (ResourceClaims against a DeviceClass such as `gpu.nvidia.com`) or a device plugin (extended resources such as `nvidia.com/gpu`). Make sure your nodes allow testing pods to be scheduled on them (e.g. no taints that prevent scheduling).
- Network Access: The test machine must be able to reach the Kubernetes API server.

## Running the Tests

Run the tests using the standard go test command. You should run tests with the same release tag as your cluster version, to ensure compatibility with the Kubernetes AI conformance program.

By default, the test looks for your kubeconfig at `~/.kube/config`. You can override this by setting the `KUBECONFIG` environment variable or using the `-kubeconfig` flag.

```bash
export KUBECONFIG=/path/to/my/config # Optional
go test -v ./test [-run <TestName>] [-kubeconfig=<path/to/kubeconfig>] [-accelerator-type=<type>] [-allocation-mode=<mode>]
```

Run `go test ./test -args -help` for details about each flag.

The allocation-mode detection, pod-construction, and device-probe logic also has hermetic unit tests that need no cluster (Kubernetes API tests use a fake clientset):

```bash
go test -v ./test -run 'Test(DetectAllocationMode|ExtendedResourceGuardFailsClosed|LookupAcceleratorConfig|BuildTestPod|PodGeneratedClaims|DeleteAndAwaitRelease|AcceleratorProbeCommand|LogsContainExactLine)$'
```

### Test Cases Covered

| Test Name | Requirement Covered | Requirement Level |
|-|-|-|
| `TestSecureAcceleratorAccess` | Secure Accelerator Access | MUST |
| `TestGangScheduling` | Gang Scheduling | MUST |

## Vendor Customization & Neutrality

The tests are designed to be vendor-neutral where possible, but hardware-level probing often requires vendor-specific configuration. If your platform uses different hardware/software not covered by the tests, please file an issue to request support for your hardware/software. In the meantime, you will need to certify manually.

### Opting Out

If a specific test is not applicable to your platform (i.e., you answered "N/A" to the corresponding question in the questionnaire), you may "opt-out" of that specific sub-test.

## Automation & CI

For CI environments, you can output the results in machine-readable JSON format, which can be converted to JUnit/XML for reporting.

```bash
go test -v ./test -json > results.json
```
