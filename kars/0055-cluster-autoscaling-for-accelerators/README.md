# KAR-0055: Effective Cluster Autoscaling for Accelerators

## Description

If the platform provides a cluster autoscaler or an equivalent mechanism, it must be able to scale up/down node groups containing specific accelerator types based on pending pods requesting those accelerators.

## Motivation

Accelerators like GPUs are expensive and scarce resources. When AI/ML workloads request accelerators that are not currently available in the cluster, the platform must be able to automatically provision new nodes with the appropriate accelerator types. Conversely, when accelerator nodes are idle, they should be scaled down to avoid unnecessary cost.

Without cluster autoscaler awareness of accelerator resources, pending workloads may wait indefinitely for resources that could be provisioned automatically, or idle accelerator nodes may continue running and incurring cost.

## Graduation Criteria

**SHOULD**
- [x] Describe how users can test it for self-attestation with scripts, documentation, etc
- [ ] Starting v1.37, new SHOULDs must include proposed automated tests in the automated tests section below

**MUST**
- [ ] Starting v1.37, new MUSTs must include automated tests that have been added to the AI conformance test suite
- [x] Demonstrate at least two real-world usage of SHOULD before graduating to MUST
- [x] Kubernetes core APIs must be GA

## Test Plan

### How We Might Test It

Prepare a node pool with N nodes, configured with a specific accelerator type, with min node pool size of N and max size of at least N+1. Assuming each node provides exactly 1 accelerator of the specified type, create N+1 Pods, each requesting one accelerator resource from that pool. Verify that at least one Pod is unschedulable (Pending), and that the cluster autoscaler subsequently increases the node count to N+1, allowing the pending Pod to transition to Running. Delete that Pod; verify that the cluster autoscaler removes the idle accelerator node, returning the node count to N.

### Automated Tests

The test verifies that a cluster autoscaler running on the platform scales node groups up and down based on pending pods requesting accelerator resources.

1. **Setup**: Configure a node pool with N nodes, each providing exactly 1 accelerator of type A, with a minimum size of N and a maximum size of at least N+1.

2. **Scale-up case**: Create N+1 Pods, each requesting one accelerator of type A from the pool. Verify that at least one Pod is initially `Pending`, the node count grows to N+1, and the Pending Pod transitions to `Running`.

3. **Scale-down case**: Delete the workload that triggered the scale-up. Verify that the autoscaler removes the now-idle accelerator node, returning the pool to its minimum size N.

4. **Cleanup**: Delete remaining test resources and restore the node pool to its original configuration.

This is implemented by `TestAcceleratorClusterAutoscaling` in the AI conformance test suite (see [test/cluster_autoscaling_test.go](../../test/cluster_autoscaling_test.go)). The test uses a provider-supplied node-pool label and required Pod anti-affinity to place one accelerator-requesting Pod on each baseline node. It validates device-plugin capacity directly and uses isolated baseline allocations plus a Pending trigger to validate effective DRA capacity. The pool must have no competing workloads or accelerator demand. The additional trigger Pod carries neither the baseline run label nor anti-affinity, so exhausted accelerator capacity is its only expected unsatisfied target-pool constraint. The test verifies that the selected pool gains Ready capacity, the Pod runs on a Node UID outside the stable baseline, and the pool returns to the original Ready, schedulable baseline after the Pod is deleted. A retained NotReady Node object does not fail scale-down because Kubernetes does not provide a portable cloud node-group size API. The test observes a preconfigured pool and does not modify provider node-pool settings.

## Implementation History

2026-03-12: KAR created

## Related KARs

<!--
List KARS that are related. This is in case of additional requirements that come up after a KAR has already graduated to "implemented"
-->
