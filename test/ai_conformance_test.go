package conformance

import (
	"context"
	"flag"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		if err != nil {
			t.Errorf("Failed to cleanup namespace: %v", err)
		}
	})

	checkDRA(ctx, t, clientset)
	setupTestEnvironment(ctx, t, clientset, namespace)

	// Getting an accelerator from inside a Pod that requests an accelerator should succeed
	t.Run("PositiveAccessTest", func(t *testing.T) {
		podName := "pos-pod"
		claims := []corev1.PodResourceClaim{{
			Name:   "claim",
			ResourceClaimTemplateName: &testResourceTemplateName,
		}}
		t.Cleanup(func() {
			clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
		})
		runPodWithClaim(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("prober")}, claims)
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "prober", true)
	})

	// Getting an accelerator from inside a Pod that does not request an accelerator should fail
	t.Run("NegativeIsolationTest", func(t *testing.T) {
		podName := "neg-pod"
		t.Cleanup(func() {
			clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
		})
		runPodWithClaim(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("prober")}, nil)
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "prober", false)
	})

	// Getting an accelerator from another container inside a Pod should fail
	t.Run("MultiContainerIsolationTest", func(t *testing.T) {
		podName := "multi-container-pod"
		claims := []corev1.PodResourceClaim{{
			Name:   "claim",
			ResourceClaimTemplateName: &testResourceTemplateName,
		}}
		t.Cleanup(func() {
			clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
		})
		runPodWithClaim(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("authorized"), acceleratorProbingContainer("unauthorized")}, claims)

		// The first container can access the accelerator, the second cannot
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "authorized", true)
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "unauthorized", false)
	})
}