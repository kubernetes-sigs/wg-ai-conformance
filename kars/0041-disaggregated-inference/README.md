# KAR-0041: Disaggregated Inference Support

## Description

Support the deployment and operation of disaggregated inference architectures. To be conformant, the platform should demonstrate that it can successfully install and run a disaggregated inference solution (e.g., vLLM, SGLang, llm-d, or Dynamo, with separate instances for distinct phases like prefill and decode). Disaggregated serving splits phases such as prefill and decode into separately scalable components so each phase can match different compute/memory/network needs, improving GPU utilization and tail latency while enabling higher throughput under mixed and bursty LLM workloads.

## Motivation

Disaggregated serving is becoming a standard architectural pattern for efficient Large Language Model (LLM) serving. By splitting prefill and decode phases, operators can optimize hardware utilization (e.g., compute-bound vs. memory-bound phases) and improve latency. Inference backends like vLLM, SGLang, TensorRT-LLM , etc and Kubernetes frameworks like llm-d, Dynamo, etc facilitate this modern architecture, which depends on Kubernetes scheduling/topology controls and strong east‑west connectivity, because intermediate decode state (e.g., KV/cache blocks and coordination metadata) must be exchanged between pods with low overhead, especially when they are placed on different nodes.

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

First, deploy a gateway provider. Then, deploy a disaggregated inference solution and run a model that supports disaggregated inference on it. Expose the gateway and send a test inference request. Finally, verify that the inference request is successful.

In addition, verify that the components are correctly disaggregated by ensuring that the distinct phases (e.g., prefill and decode) are running in separate pods, and that intermediate state (like KV cache) is successfully transferred across pods. This can be validated by checking that the same request ID (or correlation ID) appears in the logs of the separate pods handling the distinct phases.

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
