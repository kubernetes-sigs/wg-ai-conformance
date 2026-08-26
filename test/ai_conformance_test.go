package conformance

import (
	"context"
	"flag"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

// TestSecureAcceleratorAccess verifies the Secure Accelerator Access requirement.
// A Pod without an accelerator request must NOT see device nodes or have access to drivers.
// Ref: https://github.com/cncf/k8s-ai-conformance/blob/main/docs/AIConformance-1.35.yaml#L83-L89
func TestSecureAcceleratorAccess(t *testing.T) {
	if !flag.Parsed() {
		flag.Parse()
	}

	clientset := getClientset(t)

	ctx := context.Background()
	namespace := randomNamespaceName("ai-conformance")

	t.Cleanup(func() {
		if err := deleteNamespaceAndWait(ctx, t, clientset, namespace); err != nil {
			t.Errorf("CLEANUP FAILURE: %v. Please ensure this namespace is terminated manually to avoid resource leaks.", err)
		}
	})

	// The requirement permits accelerator access mediated by "device plugin
	// or DRA"; resolve which mechanism to exercise on this cluster.
	env := resolveAllocationMode(ctx, t, clientset)
	mode, cfg := env.mode, env.cfg
	setupTestEnvironment(ctx, t, clientset, namespace, mode, cfg)

	// acceleratorNode is resolved at parent level (from the environment
	// probe) so each subtest stays self-contained under -run filtering; the
	// positive pod's actual placement refines it when that subtest runs. The
	// subtests below run sequentially in declaration order (none call
	// t.Parallel), so the refinement is ordered before its use.
	acceleratorNode := env.acceleratorNode

	// A container that requests one accelerator should see exactly one.
	t.Run("PositiveAccessTest", func(t *testing.T) {
		podName := "pos-pod"
		var pod *corev1.Pod
		t.Cleanup(func() {
			if err := deletePodAndWait(ctx, clientset, namespace, podName, pod); err != nil {
				t.Errorf("Cleanup of Pod %s incomplete; subsequent accelerator tests may race its device/claim release: %v", podName, err)
			}
		})
		pod = runTestPod(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("prober", cfg)},
			testPodConfig{grantAccelerator: true, mode: mode, cfg: cfg})
		acceleratorNode = pod.Spec.NodeName
		verifyAcceleratorCountInLogs(ctx, t, clientset, namespace, podName, "prober", requestedAcceleratorCount)
	})

	// A container that does not request an accelerator should see none.
	// The pod is pinned to the accelerator node the positive pod ran on — an
	// unpinned request-less pod could schedule onto a CPU-only node and pass
	// vacuously without proving isolation.
	t.Run("NegativeIsolationTest", func(t *testing.T) {
		if acceleratorNode == "" {
			// Defensive: detection always returns a concrete node on
			// success, so this state should be unreachable. Fail rather
			// than skip — an unpinned request-less pod could pass vacuously
			// on a CPU-only node.
			t.Fatal("BUG: no accelerator node identified despite successful allocation-mode detection; cannot pin the request-less pod")
		}
		podName := "neg-pod"
		var pod *corev1.Pod
		t.Cleanup(func() {
			if err := deletePodAndWait(ctx, clientset, namespace, podName, pod); err != nil {
				t.Errorf("Cleanup of Pod %s incomplete; subsequent accelerator tests may race its device/claim release: %v", podName, err)
			}
		})
		pod = runTestPod(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("prober", cfg)},
			testPodConfig{grantAccelerator: false, mode: mode, nodeName: acceleratorNode, cfg: cfg})
		verifyAcceleratorCountInLogs(ctx, t, clientset, namespace, podName, "prober", 0)
	})

	// An accelerator granted to one container must not be visible to another.
	t.Run("MultiContainerIsolationTest", func(t *testing.T) {
		podName := "multi-container-pod"
		var pod *corev1.Pod
		t.Cleanup(func() {
			if err := deletePodAndWait(ctx, clientset, namespace, podName, pod); err != nil {
				t.Errorf("Cleanup of Pod %s incomplete; subsequent accelerator tests may race its device/claim release: %v", podName, err)
			}
		})
		pod = runTestPod(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("authorized", cfg), acceleratorProbingContainer("unauthorized", cfg)},
			testPodConfig{grantAccelerator: true, mode: mode, cfg: cfg})

		// The first container sees exactly the requested count; the second sees none.
		verifyAcceleratorCountInLogs(ctx, t, clientset, namespace, podName, "authorized", requestedAcceleratorCount)
		verifyAcceleratorCountInLogs(ctx, t, clientset, namespace, podName, "unauthorized", 0)
	})
}
