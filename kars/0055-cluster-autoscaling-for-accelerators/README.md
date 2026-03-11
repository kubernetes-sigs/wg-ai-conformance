# KAR-0055: Effective Cluster Autoscaling for Accelerators

## Description

If the platform provides a cluster autoscaler or an equivalent mechanism, it must be able to scale up/down node groups containing specific accelerator types based on pending pods requesting those accelerators.

## Motivation

Accelerators like GPUs are expensive and scarce resources. When AI/ML workloads request accelerators that are not currently available in the cluster, the platform must be able to automatically provision new nodes with the appropriate accelerator types. Conversely, when accelerator nodes are idle, they should be scaled down to avoid unnecessary cost.

Without cluster autoscaler awareness of accelerator resources, pending workloads may wait indefinitely for resources that could be provisioned automatically, or idle accelerator nodes may continue running and incurring cost.

## Graduation Criteria

**SHOULD**
- [ ] Describe how users can test it for self-attestation with scripts, documentation, etc
- [ ] Starting v1.37, new SHOULDs must include proposed automated tests in the automated tests section below

**MUST**
- [ ] Starting v1.37, new MUSTs must include automated tests that have been added to the AI conformance test suite
- [ ] Demonstrate at least two real-world usage of SHOULD before graduating to MUST
- [ ] Kubernetes core APIs must be GA

## Test Plan

### How We Might Test It

Prepare a node pool with N nodes, configured with a specific accelerator type, with min node pool size of N and max size of at least N+1. Assuming 1 accelerator A per node N, create (A*N)+1 Pods, each requesting one accelerator resource from that pool. Verify that at least one Pod is unschedulable (Pending), and the cluster autoscaler will increase the node count to N+1, causing the Pod to be Running. Delete that Pod, then the cluster autoscaler will remove the idle accelerator node, returning the node count to N.

### Automated Tests

<!--
Document all the automated tests for validating this requirement.
-->

## Implementation History

2026-03-12: KAR created

## Related KARs

<!--
List KARS that are related. This is in case of additional requirements that come up after a KAR has already graduated to "implemented"
-->
