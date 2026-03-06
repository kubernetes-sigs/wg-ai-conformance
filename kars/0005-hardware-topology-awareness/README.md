# KAR-0005: Hardware Topology Awareness

## Description

Hardware topology awareness requires the Kubernetes cluster to understand the physical layout of specialized hardware, such as GPU-GPU interconnects, GPU-NIC proximity, and more, so the scheduler can place workloads on nodes where accelerators have optimal latency and bandwidth. This ensures high-performance AI/ML jobs are scheduled on hardware configurations that maximize throughput and avoid suboptimal paths. This information should be exposed via DRA attributes if supported by accelerator and driver to enable topology-aware scheduling.

## Motivation

AI/ML workloads often depend on high-bandwidth, low-latency connections between GPUs, CPUs, and NICs, so placing workloads without awareness of hardware topology can severely degrade training and inference performance. Exposing this topology information enables Kubernetes to make intelligent, DRA-driven scheduling decisions that maximize throughput, minimize communication bottlenecks, and ensure accelerators are used to their full potential.

## Graduation Criteria

**SHOULD**
- [x] Describe how users can test it for self-attestation with scripts, documentation, etc
- [ ] Starting v1.37, new SHOULDs must include proposed automated tests in the automated tests section below

**MUST**
- [ ] Starting v1.37, new MUSTs must include automated tests that have been added to the AI conformance test suite
- [ ] Demonstrate at least two real-world usage of SHOULD before graduating to MUST
- [ ] Kubernetes core APIs must be GA

## Test Plan

### How We Might Test It

We can deploy workloads with known topology sensitivities, such as multi-GPU training jobs requiring NVLink or GPU-direct RDMA. Then verify that Kubernetes schedules them onto nodes whose accelerator topology matches the requested DRA attributes, while rejecting or rescheduling pods on incompatible nodes.

### Automated Tests

Implement an e2e test that assumes DRA (resource.k8s.io/v1) is enabled and a GPU DRA driver is installed:
1. Creates two DeviceClass objects with distinct topology attributes (e.g., requires_nvlink=true, requires_gpu_nic_proximity=true).
1. For each DeviceClass, creates a ResourceClaimTemplate and launches a test pod that references it.
1. Asserts in the test harness that `pod.spec.nodeName` satisfies the requested topology (via DRA attributes e.g. `device.attributes["dra.net"].rdma == true`)
1. Inside the pod, run a small probe script (e.g., topology_probe.sh) that verifies NVLink/RDMA presence and exits non-zero on mismatch, causing the test case to fail.

## Implementation History

2025-11-30: KAR created

## Related KARs

<!--
List KARS that are related. This is in case of additional requirements that come up after a KAR has already graduated to “implemented”
-->