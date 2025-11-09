<!--
**Note:** When your KAR is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [ ] **Create an issue in kubernetes-sigs/wg-ai-conformance**
  When filing an AI conformance requirement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KAR. You
  can leave that blank until this KAR is filed, and then go back to the
  issue and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the kars directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your AI conformance requirement issue above.
- [ ] **Fill out as much of the kar.yaml file as you can.**
  At minimum, you should fill in the "title", "kar-number", "status", "stage", "milestone", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Description" sections.
- [ ] **Create a PR for this KAR.**
  Assign it to ai-conformance-requirement-approvers to review and approve.

When editing KARS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KAR corresponds to one "AI conformance requirement" for its whole lifecycle.
You do not need a new KAR to move from SHOULD to MUST, for example. If
new details emerge that belong in the KAR, edit the KAR. Once a requirement has become
"implemented", major changes should be driven as a new KAR.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/kars/NNNN-kar-template/README.md).
-->

# KAR-NNNN: Your short, descriptive title

<!--
This is the title of your KAR. Keep it short, simple, and descriptive. A good
title can help communicate what the KAR is and should be considered as part of
any review.
-->

## Description

<!--
The CNCF Kubernetes AI Conformance defines a set of capabilities, APIs, and configurations that a Kubernetes cluster MUST offer, on top of standard CNCF Kubernetes Conformance, to reliably and efficiently run AI/ML workloads. This initiative aims to simplify AI/ML operations on Kubernetes, accelerate adoption, guarantee interoperability and portability for AI workloads, reduce the overall cost of ownership, and enable ecosystem growth on an industry-standard foundation.

This section should produce high-quality, user-focused
documentation for an AI conformance requirement that will be part of a corresponding Kubernetes release in https://github.com/cncf/k8s-ai-conformance. Vendors should be able to understand the requirement and submit conformance results for review and certification by the CNCF. A test implementer should be able to create automated tests based on this description.

A good description should be one or two sentences in length.
-->

## Motivation

<!--
This section is for explicitly listing the motivation and rationale of why the requirement is important and the benefits to users. The section can optionally provide links to existing implementations to demonstrate the interest in this KAR within the wider Kubernetes community.
-->

## Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*
If applicable, make sure the required tests are referenced in the test plan section.
-->

**SHOULD**
- [ ] Describe how users can test it for self-attestation with scripts, documentation, etc
- [ ] Starting v1.37, new SHOULDs must include proposed automated tests in the automated tests section below

**MUST**
- [ ] Starting v1.37, new MUSTs must include automated tests that have been added to the AI confromance test suite
- [ ] Demonstrate at least two real-world usage of SHOULD before graduating to MUST
- [ ] Kubernetes core APIs must be GA
<!--
**Note:** We recommend that non-core APIs should be GA as well, but it is not required.
-->

## Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept requirements with inadequate ways to test them.
Starting v1.37, new SHOULDs must include proposed automated tests and new MUSTs must include automated tests that have been added to the AI conformance test suite.
For SHOULDs, users can run automated test or self-attestation following manual steps described in How We Might Test It section below.
-->

### How We Might Test It
<!--
**Note:** *Not required until targeted at a release.*
This section should document scripts and manual steps to validate for self-attestation.
-->

### Automated Tests

<!--
**Note:** *Not required until targeted at a release.*
Document all the automated tests for validating this requirement.
-->

## Implementation History

<!--
Major milestones in the lifecycle of a KAR should be tracked in this section.
Major milestones might include:
- the date the KAR is created and its status changed to implementable, signaling WG acceptance
- the `Test Plan` section being merged, signaling agreement on a proposed test plan
- the first Kubernetes release where an initial version of the KAR was available as SHOULD
- the version of Kubernetes where the KAR graduated to MUST
- the date the status changed to implemented from implementable, signaling completion
- when the KAR was retired or superseded
-->

## Related KARs

<!--
List KARS that are related. This is in case of additional requirements that come up after a KAR has already graduated to “implemented”
-->
