## Prerequisites
To run these AI Conformance tests, you must have:

- Golang: Installed on your local machine.
- Kubeconfig: A valid kubeconfig file with cluster-admin permissions for the target cluster.
- Accelerator Node Pool: The cluster must have nodes with accelerators exposed through the Kubernetes resource management framework — either a DRA driver (ResourceClaims against a DeviceClass such as `gpu.nvidia.com`) or a device plugin (extended resources such as `nvidia.com/gpu`). Make sure your nodes allow testing pods to be scheduled on them (e.g. no taints that prevent scheduling).
- Cluster Autoscaling Test: `TestAcceleratorClusterAutoscaling` additionally requires a running cluster autoscaler and an isolated accelerator pool with minimum size `N >= 1`, maximum size at least `N+1`, effective capacity for exactly one requested accelerator per baseline node, scale-down enabled, sufficient cloud quota/stock, and one stable node label inherited by new pool nodes. The pool must contain no non-DaemonSet workloads or other Pending Pods explicitly selecting the pool. Device-plugin mode permits unrelated running accelerator workloads outside the pool but rejects other Pending Pods requesting the configured extended resource. DRA mode requires no other active Pods with ResourceClaims or allocated ResourceClaims outside the test namespace while the test runs because DRA devices may use shared topology.
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
| `TestAcceleratorClusterAutoscaling` | Effective Cluster Autoscaling for Accelerators | MUST |

### Accelerator Cluster Autoscaling

The autoscaling test observes a preconfigured accelerator pool; it does not
modify provider node-pool settings. Use a node label that uniquely identifies
the target pool. The example below uses the AKS `agentpool` label; replace it
with the corresponding label for your platform.

Baseline Pods use hostname anti-affinity to occupy distinct pool nodes. The
trigger Pod carries neither that anti-affinity nor its matching label, so
exhausted accelerator capacity is the only target-pool constraint that can
leave it Pending. A PodDisruptionBudget protects the baseline Pods while the
added node is idle and eligible for scale-down. The test fails if pool
membership changes or foreign accelerator demand appears during scale-up.

Device-plugin mode rejects extended resources backed by a DRA DeviceClass and
checks every observed pool node for exactly one allocatable extended resource.
DRA has no portable per-node allocatable count, so the isolated baseline Pods
and Pending trigger verify that the selected DeviceClass has effective capacity
for exactly `N` allocations before scale-up.

Scale-up and scale-down can take significantly longer than Go's default test
timeout, so use `-timeout 75m` or a larger value.

```bash
go test -v ./test \
  -run TestAcceleratorClusterAutoscaling \
  -timeout 75m \
  -accelerator-type=nvidia \
  -autoscaler-allocation-mode=device-plugin \
  -autoscaler-node-pool-label=agentpool=gpupool
```

To test DRA autoscaling, the cluster autoscaler must have DRA support enabled:

```bash
go test -v ./test \
  -run TestAcceleratorClusterAutoscaling \
  -timeout 75m \
  -accelerator-type=nvidia \
  -autoscaler-allocation-mode=dra \
  -autoscaler-node-pool-label=agentpool=gpupool
```

The following optional flags control observation windows:

- `-autoscaler-pending-timeout` (default `2m`)
- `-autoscaler-scale-up-timeout` (default `20m`)
- `-autoscaler-scale-down-timeout` (default `30m`)
- `-autoscaler-stability-window` (default `30s`)

Scale-down passes when the original baseline Node UIDs remain Ready and schedulable, the added
Node UID is no longer Ready or has been deleted, and the selected pool has
exactly `N` Ready Nodes for the stability window. The Kubernetes API cannot
portably prove a cloud provider's node-group size, and Cluster Autoscaler does
not itself guarantee deletion of the Kubernetes `Node` object.

## Vendor Customization & Neutrality

The tests are designed to be vendor-neutral where possible, but hardware-level probing often requires vendor-specific configuration. If your platform uses different hardware/software not covered by the tests, please file an issue to request support for your hardware/software. In the meantime, you will need to certify manually.

### Opting Out

If a specific test is not applicable to your platform (i.e., you answered "N/A" to the corresponding question in the questionnaire), you may "opt-out" of that specific sub-test.

## Automation & CI

For CI environments, you can output the results in machine-readable JSON format, which can be converted to JUnit/XML for reporting.

```bash
go test -v ./test -json > results.json
```
