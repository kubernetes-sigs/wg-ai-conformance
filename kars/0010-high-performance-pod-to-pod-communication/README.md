# KAR-0010: High-Performance Pod-to-Pod Communication

## Description

If high performance pod-to-pod communication is needed, then provide well-defined mechanisms for these specialized network resources to be managed and exposed such that their characteristics should be discoverable to enable informed scheduling or workload configuration and to enable pods to attach to multiple network interfaces.

Forward-looking: Once the network resource supports DRA, then the platform should use the DRA mechanism.

## Motivation

AI/ML workloads, particularly distributed training and inference, require high-throughput, low-latency pod-to-pod communication. These workloads often rely on specialized network hardware (e.g., RDMA-capable NICs, high-speed interconnects) that must be explicitly attached to pods. Without a standardized mechanism for discovering and allocating these network resources, users face fragmented tooling, inconsistent behavior across platforms, and difficulty ensuring workloads are scheduled on nodes with the appropriate network capabilities.

By leveraging Dynamic Resource Allocation (DRA) for network resources, platforms can provide a consistent, Kubernetes-native way to expose high-performance network interfaces to pods. This enables workloads to discover available network characteristics and make informed scheduling decisions, improving portability and reducing the operational burden of running distributed AI/ML workloads on Kubernetes.

## Graduation Criteria

**SHOULD**
- [X] Describe how users can test it for self-attestation with scripts, documentation, etc
- [ ] Starting v1.37, new SHOULDs must include proposed automated tests in the automated tests section below

**MUST**
- [ ] Starting v1.37, new MUSTs must include automated tests that have been added to the AI conformance test suite
- [ ] Demonstrate at least two real-world usage of SHOULD before graduating to MUST
- [ ] Kubernetes core APIs must be GA

## Test Plan

### How We Might Test It

Validate the following observable outcomes:

1. **Multiple network interfaces are available to pods:** A pod scheduled on a node with high-performance network hardware has access to additional network interfaces beyond the default pod network.
2. **Network resource characteristics are discoverable:** The characteristics of available high-performance network resources (e.g., interface type, bandwidth, RDMA capability) are published and queryable within the cluster, enabling workloads and schedulers to make informed decisions.
3. **Pods are scheduled to nodes with the required network resources:** When a workload requires a specific high-performance network capability, it is scheduled only on nodes where that capability is available.
4. **Pod-to-pod communication functions over the high-performance interface:** Two pods on nodes with high-performance networking can exchange data over the specialized interface, confirming end-to-end connectivity.

### Automated Tests

Automated tests should verify the outcomes above:

- Deploy a pod to a node with high-performance network hardware and confirm that the pod's network namespace contains the expected additional network interface(s).
- Query the cluster for published network resource characteristics and validate that they accurately describe the available high-performance network capabilities.
- Deploy a workload requesting a specific network capability and verify it is scheduled on an appropriate node.
- Deploy two pods with access to high-performance network interfaces and verify successful pod-to-pod data transfer over those interfaces.

## Implementation History

2026-02-22: KAR created

## Related KARs

<!--
List KARS that are related. This is in case of additional requirements that come up after a KAR has already graduated to "implemented"
-->
