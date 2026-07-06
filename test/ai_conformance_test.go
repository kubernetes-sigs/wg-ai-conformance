package conformance

import (
	"context"
	"flag"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// TestSecureAcceleratorAccess verifies the Secure Accelerator Access requirement.
// A Pod without an accelerator request must NOT see device nodes or have access to drivers.
// Ref: https://github.com/cncf/k8s-ai-conformance/blob/main/docs/AIConformance-1.35.yaml#L83-L89
func TestSecureAcceleratorAccess(t *testing.T) {
	if !flag.Parsed() {
		flag.Parse()
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if *kubeconfig != "" {
		loadingRules.ExplicitPath = *kubeconfig
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		t.Fatalf("Error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Error creating kubernetes client: %v", err)
	}

	ctx := context.Background()
	namespace := randomNamespaceName("ai-conformance")

	t.Cleanup(func() {
		t.Logf("Cleaning up namespace %s...", namespace)
		err := clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// The test failed before the namespace was created; nothing to clean up.
			return
		}
		if err != nil {
			t.Errorf("Failed to cleanup namespace: %v", err)
		}

		// Poll until the namespace is actually gone; this is needed because the namespace needs to release resources for rerunning tests
		err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
			_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			t.Errorf("CLEANUP FAILURE: Failed to delete namespace %s: %v. "+
				"Please ensure this namespace is terminated manually to avoid resource leaks."+
				"Rerunning the tests might fail if the namespace is not deleted.",
				namespace, err)
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

	// Getting an accelerator from inside a Pod that requests an accelerator should succeed
	t.Run("PositiveAccessTest", func(t *testing.T) {
		podName := "pos-pod"
		var pod *corev1.Pod
		t.Cleanup(func() {
			deletePodAndWait(ctx, t, clientset, namespace, podName, pod)
		})
		pod = runTestPod(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("prober", cfg)},
			testPodConfig{grantAccelerator: true, mode: mode, cfg: cfg})
		acceleratorNode = pod.Spec.NodeName
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "prober", true)
	})

	// Getting an accelerator from inside a Pod that does not request an accelerator should fail.
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
			deletePodAndWait(ctx, t, clientset, namespace, podName, pod)
		})
		pod = runTestPod(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("prober", cfg)},
			testPodConfig{grantAccelerator: false, mode: mode, nodeName: acceleratorNode, cfg: cfg})
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "prober", false)
	})

	// Getting an accelerator from another container inside a Pod should fail
	t.Run("MultiContainerIsolationTest", func(t *testing.T) {
		podName := "multi-container-pod"
		var pod *corev1.Pod
		t.Cleanup(func() {
			deletePodAndWait(ctx, t, clientset, namespace, podName, pod)
		})
		pod = runTestPod(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("authorized", cfg), acceleratorProbingContainer("unauthorized", cfg)},
			testPodConfig{grantAccelerator: true, mode: mode, cfg: cfg})

		// The first container can access the accelerator, the second cannot
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "authorized", true)
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "unauthorized", false)
	})
}
