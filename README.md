# wg-ai-conformance

Proposals and discussions for the [AI Conformance Working Group](https://github.com/kubernetes/community/tree/master/wg-ai-conformance).

## Defining AI Conformance Requirements

Discussions for AI conformance requirements are now tracked using the
[WG AI Conformance Requirements](https://github.com/orgs/kubernetes-sigs/projects/114)
GitHub Project, shifting from the original
[Google Doc](https://docs.google.com/document/d/1hXoSdh9FEs13Yde8DivCYjjXyxa7j4J8erjZPEGWuzc/edit?tab=t.0).

Each requirement is tracked as a GitHub issue with the following workflow:
- **Proposed**: Requirements currently under discussion.
- **Implementable for K8s Release**: Requirements that have been accepted by the working group.
- **Implemented**: Requirements that have been added to the [cncf/ai-conformance](https://github.com/cncf/ai-conformance) repository.

To participate, please comment on the relevant GitHub issues and pull requests.
Unresolved items will be discussed during the
[recurring WG meeting](https://github.com/kubernetes/community/tree/master/wg-ai-conformance#meetings).

### Process Details

This process adopts the Kubernetes Enhancement Proposal (KEP) process as the basis for managing the lifecycle of Kubernetes AI conformance Requirements (KARs), including review, discussion, and approval.

- **Requirement Proposal**: Propose each new Kubernetes AI conformance Requirement (KAR) as a GitHub issue, targeting a specific Kubernetes release. Requirements will be tracked and reviewed individually. This is similar to a KEP, but tracked as a KAR with an issue in this wg-ai-conformance repo.
- **Graduation Criteria**: Requirements will graduate from "SHOULD" to "MUST" stage in alignment with the process used for KEPs in each Kubernetes release cycle.
- **Timeline Alignment**: The lifecycle for requirements will follow the Kubernetes release schedule: e.g. https://github.com/kubernetes/sig-release/blob/master/releases/release-1.35/README.md#timeline This timeline will be adopted starting Kubernetes v1.36.
  - **KEP/KAR Freeze**: Locks in the set of KARs to be considered for updates for a given Kubernetes release. No new requirements after KEP freeze.
  - **Discussion and Refinement before Code Freeze**: After KEP/KAR freeze, all discussions, text refinement, any changes (including associated tests) for accepted requirements will happen as part of the PR review for KAR updates. All changes for all KARs must be locked in before the code freeze deadline for the Kubernetes release.
  - **Post Code Freeze**: All changes for all KARs will be submitted to a formal list of AI conformance requirements for that given Kubernetes release in [cncf/ai-conformance](https://github.com/cncf/ai-conformance) to ensure transparency and clarity for the entire community. In the event a kubernetes feature does not reach GA and impacts the graduation of a KAR, we will need to reassess that KAR to rollback.
- **Reviewers**: everyone in wg-ai-conformance
- **Approvers**: wg-ai-conformance leads
- **Stage**: All requirements need to start with SHOULD and eventually graduate to MUST
- **Requirement removal**: Criteria to remove a requirement is TBD


## Designing AI Conformance Tests

Once there is community consensus on a requirement, the next step is to define how to verify it.
For every MUST requirement, a corresponding test will be designed. Starting v1.37, automated tests are prerequisites for new MUSTs. This test design should be documented as part of the KAR and should be specific enough to be implemented in an automated fashion.

Discussions for how we design AI conformance tests are tracked using the
[WG AI Conformance Tests Design](https://github.com/orgs/kubernetes-sigs/projects/118)
GitHub Project.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack channel](https://kubernetes.slack.com/messages/wg-ai-conformance)
- [Mailing List](https://groups.google.com/a/kubernetes.io/g/wg-ai-conformance)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
