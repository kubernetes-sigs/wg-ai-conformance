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
  Copy this template into the owning directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your AI conformance requirement issue above.
- [ ] **Fill out as much of the kar.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Description" sections.
- [ ] **Create a PR for this KAR.**
  Assign it to wg-ai-conformance leads who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KAR clarified and merged quickly. The best way to do this is to 
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KAR is merged does not mean it is complete or approved. Any KAR
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KARS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KAR corresponds to one "AI conformance requirement" for its whole lifecycle.
You do not need a new KAR to move from SHOULD to MUST, for example. If
new details emerge that belong in the KAR, edit the KAR. Once a requirement has become
"implemented", major changes should get new KARs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/kars/NNNN-kar-template/README.md).

**Note:** Any PRs to move a KAR to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KAR approvers.
If none of those approvers (wg-ai-conformance leads) are still appropriate, 
then changes to that list should be approved by the remaining approvers and/or
SIG Architecture.
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
documentation for an AI conformance requirement that will be part of a corresponding Kubernetes Release in https://github.com/cncf/ai-conformance. Vendors should be able to understand the requirement and submit conformance results for review and certification by the CNCF. A test implementer should be able to create automated tests based on this description.
KAR editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good description should be one or two sentences in length.
-->

## Motivation

<!--
This section is for explicitly listing the motivation and rationale of why the requirement is important and the benefits to users. The section can optionally provide links to existing implementations to demonstrate the interest in this KAR within the wider Kubernetes community.
-->

## Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*
If applicable, make sure the required tests are in the test plan section.
-->

**SHOULD**
- [ ] Describe how users can test it for self-attestation with scripts, documentation, etc

**MUST**
- [ ] Automated tests for this requirement must be part of the AI confromance test suite
- [ ] Demonstrate at least two real-world usage of SHOULD before graduating to MUST
- [ ] Kubernetes core APIs must be GA

## Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept requirements with inadequate ways to test them.
Starting v1.37, automated tests are graduation criteria for MUSTs.
-->

### How We Might Test It
<!--
**Note:** *Not required until targeted at a release.*
For SHOULD, describe what tests will be added to the AI conformance test suite. 
Document scripts or steps a user can follow to test for self-attestation.
-->

### Automated Tests

<!--
**Note:** *Not required until targeted at a release.*
Document automated tests that have been added to the AI conformance test suite.
-->

## Implementation History

<!--
Major milestones in the lifecycle of a KAR should be tracked in this section.
Major milestones might include:
- the `Description` and `Motivation` sections being merged, signaling WG acceptance
- the `Test Plan` section being merged, signaling agreement on a proposed test plan
- the date the status changed to implementable from provisional
- the first Kubernetes release where an initial version of the KAR was available as SHOULD
- the version of Kubernetes where the KAR graduated to MUST
- the date the status changed to implemented from implementable
- when the KAR was retired or superseded
-->

## Drawbacks

<!--
Why should this KAR _not_ be implemented?
-->
