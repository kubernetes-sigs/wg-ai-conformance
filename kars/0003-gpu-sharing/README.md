# KAR-0003: GPU Sharing

## Description

For accelerators that support GPU sharing, provide well-defined mechanisms for GPU sharing to improve utilization for workloads that do not require a full dedicated GPU. If hardware-level partitioning is supported, these fractional GPU resources should be exposed as distinct schedulable resources. If software-based sharing (e.g. time-slicing) is supported, then oversubscription of GPUs should be allowed. Once the accelerator supports these features as part of DRA, then the platform should use the DRA mechanism.

## Motivation

AI/ML workloads on Kubernetes need GPU sharing because it dramatically improves GPU utilization. Most inference and fine-tuning jobs don’t need an entire modern GPU. Partitioning lets multiple workloads run concurrently without wasting compute. It also reduces cost by allowing teams to pack more jobs onto the same hardware, and enables resource isolation so different users or namespaces can safely share the same GPU without interfering with each other. GPU sharing also provides scheduling flexibility, letting Kubernetes treat GPUs more like fine-grained, composable resources rather than monolithic devices.

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

We should be able to use the verifiable mechanism provided by the platform to ensure that a single physical GPU can be exposed as multiple allocatable GPU resources and scheduled by Kubernetes. Create functional tests that verify multiple workloads requesting multiple logical GPUs can be scheduled and are running on the same physical GPU. Another test to ensure over allocating logical GPUs is blocked. 

### Automated Tests

An automated test could involve deploying ResourceClaimTemplate and ResourceClaim for shared logical GPUs, pods using the ResourceClaimTemplate get scheduled onto the same node as all the ResourceClaim allocations, and all pods are running on that node. The pod would execute a script that inspects the accelerator health and no errors returned.

## Implementation History

2025-11-19: KAR created

## Related KARs

<!--
List KARS that are related. This is in case of additional requirements that come up after a KAR has already graduated to “implemented”
-->