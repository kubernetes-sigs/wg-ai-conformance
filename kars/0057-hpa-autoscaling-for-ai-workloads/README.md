# KAR-0057: Effective HPA Autoscaling for AI Workloads

## Description

If the platform supports the HorizontalPodAutoscaler, it must function correctly for pods utilizing accelerators. This includes the ability to scale these Pods based on custom metrics relevant to AI/ML workloads.

## Motivation

AI inference workloads often need to scale horizontally based on demand — for example, scaling up GPU-backed serving pods when request latency increases or queue depth grows. The HorizontalPodAutoscaler (HPA) is the standard Kubernetes mechanism for this, but it must work correctly with pods that consume accelerator resources and scale based on custom metrics (e.g. accelerator utilization, request throughput) rather than just CPU and memory.

Without HPA support for accelerator-backed pods and custom metrics, operators must manually scale inference workloads or build custom autoscaling solutions, which adds complexity and reduces reliability.

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

A custom metrics pipeline is configured to expose accelerator-related custom metrics to the HPA. Create a Deployment with each Pod requesting an accelerator and exposing a custom metric. Create a HorizontalPodAutoscaler targeting the Deployment. Introduce load to the sample application, causing the average custom metric value to significantly exceed the target, triggering a scale up. Then remove the load to trigger a scale down.

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
