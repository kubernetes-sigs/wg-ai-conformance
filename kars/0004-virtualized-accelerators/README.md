# KAR-0004: Virtualized Accelerators

## Description

For accelerators that support virtualized accelerator technologies (e.g. vGPU), provide well-defined mechanisms for these to be exposed and managed, maintaining consistency with physical fractional GPUs. 

Forward-looking: Once the accelerator supports virtualized accelerator technologies as part of DRA, then the platform should use the DRA mechanism.

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

We can test this capability by deploying workloads that request physical fractional GPUs and vGPUs through the same DRA mechanism, verifying that the scheduler consistently allocates, isolates, and accounts for them using identical semantics.

### Automated Tests

An automated test would involve the following steps:
1. Deploying test pods requesting one or more vGPUs e.g. 
```yaml
resources:
  limits:
    "nvidia.com/gpu": 1 # or vendor.com/vfio-device: "1" 
```
1. The pods should succesfully be running on a node with the vGPUs.
1. The pods would execute a script that inspects the accelerator health and no errors returned.

The automated tests will default to checking common accelerators (~80% of platforms). If the test encounters an accelerator variant it does not recognize, it will output an "Unknown" status rather than failing, signaling that manual verification is required. We expect the test suite to grow over time to support automated verification for all platforms.

## Implementation History

2025-11-30: KAR created

## Related KARs

<!--
List KARS that are related. This is in case of additional requirements that come up after a KAR has already graduated to “implemented”
-->