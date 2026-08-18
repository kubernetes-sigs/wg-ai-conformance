# Conformance Versions

Kubernetes AI Conformance follows Kubernetes Releases. Therefore the corresponding Version is always towards the Kubernetes Version e.g `KubernetesAIConformance-1.33.yaml` corrolates to `Kubernetes v1.33`

In this folder are the Conformance requirements for each Kubernetes Version supported by the Kubernetes AI Conformance Program. 

The files in this Folder are the final Version and will be transfered to the [cncf/k8s-ai-conformance](https://github.com/cncf/k8s-ai-conformance) repository. 

## Hybrid Verification Submission (v1.37+)

Starting with v1.37, we support a hybrid verification approach that combines automated test results with manual attestation:

1.  **Automated Tests**: For requirements with automated tests (e.g., Secure Accelerator Access, Gang Scheduling), run the test suite and include the output files (`e2e.log`, `junit.xml`) in your submission PR.
2.  **Manual Attestation**: For requirements without automated tests, continue to provide documentation or reference URLs in the `evidence` field of the checklist YAML.

Reference your automated test results in the checklist YAML using the `file://` scheme or relative paths to your submitted test artifacts.