# KAR-0004: Virtualized Accelerators

## Description

If accelerator supports virtualized accelerator technologies (e.g. vGPU), provide well-defined mechanisms for  these to be exposed and managed (via DRA once that is supported), maintaining consistency with physical fractional GPUs.

## Motivation

AI/ML workloads on Kubernetes need virtualized accelerators because they allow GPU capacity to be flexibly carved out and delivered even in virtualized or multi-tenant environments where physical devices aren’t directly assignable. Maintaining consistency with physical fractional GPUs ensures that workloads, scheduling, policies, and user expectations remain the same across both physical and virtual backends, enabling portability and avoiding fragmented, vendor-specific resource models.

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

We can test this capability by deploying workloads that request physical fractional GPUs and vGPUs through the same DRA, verifying that the scheduler consistently allocates, isolates, and accounts for them using identical semantics.

### Automated Tests

Implement an e2e test that assumes DRA (resource.k8s.io/v1) is enabled and a GPU DRA driver is installed:
1. Creates two DeviceClass objects, e.g. dc-physical-gpu and dc-vgpu, and two ResourceClaimTemplates that each reference one of these.
1. For each template, instantiates 2 almost identical test pods that consume the claims via ResourceClaimTemplates.
1. In the test harness, asserts that all pods become Running, that pod.spec.nodeName matches nodes advertising the expected device type (physical vs vGPU via DRA attributes), and that per-pod health-check scripts (e.g. accelerator_probe.sh) exit with code 0.
1. Collects and compares basic metrics (allocated device count, visible memory, isolation checks like unique device IDs) between the physical and virtual variants to confirm that allocation, isolation, and accounting semantics are consistent without any change to the workload spec other than which template it references.

## Implementation History

2025-11-30: KAR created

## Related KARs

<!--
List KARS that are related. This is in case of additional requirements that come up after a KAR has already graduated to “implemented”
-->