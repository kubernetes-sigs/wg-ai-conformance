# KAR-0063: Robust CRD and Controller Operation

## Description

The platform must prove that at least one complex AI operator with a CRD (e.g., Ray, Kubeflow) can be installed and functions reliably. This includes verifying that the operator's pods run correctly, its webhooks are operational, and its custom resources can be reconciled.

## Motivation

AI/ML workloads on Kubernetes are typically managed by operators that extend the platform with custom resources, such as RayCluster, PyTorchJob, or InferenceService. These operators rely on CRDs, admission webhooks, and controller reconciliation loops to function correctly. If any of these components fail silently or are incompatible with the platform, AI workloads cannot be deployed or managed reliably.

This requirement ensures the platform can support the full lifecycle of at least one representative AI operator, providing confidence that the Kubernetes API extensions ecosystem works correctly for AI use cases.

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

Deploy a representative AI operator, verify all Pods of the operator and its webhook are Running and its CRDs are registered with the API server. Verify that invalid attempts (e.g. invalid spec) should be rejected by its admission webhook. Verify that a valid instance of the custom resource can be reconciled.

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
