# KAR-0041: Disaggregated Inference Support

## Description

Support the deployment and operation of disaggregated inference architectures. To be conformant, the platform should demonstrate that it can successfully install and run a disaggregated inference solution (e.g., vLLM with separate prefill/decode instances, llm-d, or Dynamo). This ensures the platform provides the necessary networking, scheduling, and hardware capabilities to support modern disaggregated inference workloads.

## Motivation

Disaggregated serving is becoming a standard architectural pattern for efficient Large Language Model (LLM) serving. By splitting prefill and decode phases, operators can optimize hardware utilization (e.g., compute-bound vs. memory-bound phases) and improve latency. Frameworks like vLLM, llm-d, and Dynamo facilitate this modern architecture, but they require robust underlying Kubernetes scheduling and high-bandwidth, low-latency networking to seamlessly transfer the K/V cache between pods.

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

We can verify this capability by deploying an open-source disaggregated inference framework and verifying that it runs and works as expected. 

First, deploy a gateway provider. Then, deploy a disaggregated inference solution. Expose the gateway and send a test inference request. Finally, verify that the inference request is successful.

### Automated Tests

<!--
**Note:** *Not required until targeted at a release.*
Document all the automated tests for validating this requirement.
-->

## Implementation History

2026-02-20: KAR created

## Related KARs

<!--
List KARS that are related. This is in case of additional requirements that come up after a KAR has already graduated to “implemented”
-->
