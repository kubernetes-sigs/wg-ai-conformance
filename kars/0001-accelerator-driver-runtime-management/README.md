# KAR-0001: Accelerator Driver & Runtime Management

## Description

Provide a verifiable mechanism for ensuring that compatible accelerator drivers and corresponding container runtime configurations are correctly installed and maintained on nodes with accelerators.

## Motivation

AI/ML workloads often have strict dependencies on specific versions of accelerator drivers and container runtimes. Incompatibility between these components is a common source of errors, leading to wasted resources and developer frustration. 

This requirement ensures that a Kubernetes platform provides a reliable way to verify that the correct accelerator drivers and container runtimes are installed and configured on nodes with accelerators.

By providing a verifiable mechanism to ensure compatibility, platforms can significantly improve the reliability of AI/ML workloads, simplify troubleshooting, and guarantee portability across different environments. This helps accelerate the adoption of AI/ML on Kubernetes by providing a more stable and predictable platform.

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

We should be able to use the verifiable machenism provided by the platform to ensure that the compatible accelerator driver and container runtime are installed and configured correctly on nodes with accelerators. This might be achieved by using labels and annotations on the nodes, or through DRA attributes.

### Automated Tests

An automated test could involve deploying a pod to a node with a specific accelerator type. The pod would execute a script that inspects the node's environment to determine the actual installed driver and runtime versions. This should be done by leveraging the verifiable mechanism provided by the platform.

## Implementation History

2025-11-19: KAR created

## Related KARs

<!--
List KARS that are related. This is in case of additional requirements that come up after a KAR has already graduated to “implemented”
-->
