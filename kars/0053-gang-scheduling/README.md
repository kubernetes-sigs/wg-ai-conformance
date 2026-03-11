# KAR-0053: Gang Scheduling

## Description

The platform must allow for the installation and successful operation of at least one gang scheduling solution that ensures all-or-nothing scheduling for distributed AI workloads (e.g. Kueue, Volcano, etc.) To be conformant, the vendor must demonstrate that their platform can successfully run at least one such solution.

## Motivation

Distributed AI workloads such as multi-node training jobs require all of their component pods to be scheduled simultaneously. If only some pods in a group are scheduled while others remain pending, the running pods waste expensive accelerator resources while waiting for the rest. Gang scheduling ensures that either all pods in a job are placed at once, or none are, preventing resource deadlocks and wasted capacity.

This is especially critical for large-scale training jobs that may require dozens or hundreds of coordinated pods across multiple nodes with accelerators.

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

Install a gang scheduling solution (e.g. Kueue or Volcano) on the cluster. Submit a distributed AI workload that requires multiple pods to be co-scheduled. Verify that all pods are scheduled simultaneously and the workload completes successfully. Then submit a workload that requests more resources than available and verify that no partial scheduling occurs — all pods should remain pending until sufficient resources are available.

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
