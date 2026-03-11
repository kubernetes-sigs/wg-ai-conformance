# KAR-0059: Accelerator Performance Metrics

## Description

For supported accelerator types, the platform must allow for the installation and successful operation of at least one accelerator metrics solution that exposes fine-grained performance metrics via a standardized, machine-readable metrics endpoint. This must include a core set of metrics for per-accelerator utilization and memory usage. Additionally, other relevant metrics such as temperature, power draw, and interconnect bandwidth should be exposed if the underlying hardware or virtualization layer makes them available. The list of metrics should align with emerging standards, such as OpenTelemetry metrics, to ensure interoperability. The platform may provide a managed solution, but this is not required for conformance.

## Motivation

Observability into accelerator health and performance is fundamental for operating AI/ML workloads reliably. Without fine-grained metrics, operators cannot detect underutilized GPUs, thermal throttling, memory pressure, or failing hardware. These scenarios may directly impact training throughput and inference latency.

Standardizing the metrics format and ensuring at least one solution is operational on the platform allows consistent tooling across different vendors and accelerator types, reducing the operational burden of running AI workloads at scale.

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

Given a node with a supported accelerator type, identify the Prometheus-compatible metrics endpoint for the accelerators on the node and scrape metrics from the endpoint. Parse the scraped metrics to find metrics for each supported accelerator on the node, including: accelerator utilization, memory usage, temperature, power usage, etc. The test can evolve to check for specific metric names once those are standardized.

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
