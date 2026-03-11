# KAR-0061: AI Job & Inference Service Metrics

## Description

Provide a monitoring system capable of discovering and collecting metrics from workloads that expose them in a standard format (e.g. Prometheus exposition format). This ensures easy integration for collecting key metrics from common AI frameworks and servers.

## Motivation

AI frameworks and inference servers commonly expose operational metrics in Prometheus format, including request latency, throughput, queue depth, batch sizes, and more. Without a platform-level monitoring system that can automatically discover and collect these metrics, operators must manually configure metric collection for each workload, which is error-prone and does not scale.

This requirement ensures the platform provides a monitoring system that can discover workloads exposing standard metrics and collect them, enabling consistent observability across AI workloads regardless of which framework they use.

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

Because all these common metrics are exposed in the Prometheus format, the test verifies the platform’s monitoring system can collect Prometheus metrics. First deploy an AI application using a common framework, configure metrics collection for this application, generate sample traffic to the application, then queries the platform's monitoring system and verifies that key metrics from the AI application have been collected.

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
