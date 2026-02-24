package conformance

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var kubeconfig *string

func init() {
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "absolute path to the kubeconfig file")
	}
}

// TestSecureAcceleratorAccess verifies the Secure Accelerator Access requirement.
// A Pod without an accelerator request must NOT see device nodes or have access to drivers.
// Ref: https://github.com/cncf/k8s-ai-conformance/blob/main/docs/AIConformance-1.35.yaml#L83-L89
func TestSecureAcceleratorAccess(t *testing.T) {
	if !flag.Parsed() {
		flag.Parse()
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		t.Fatalf("Error building kubeconfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Error creating kubernetes client: %v", err)
	}

	ctx := context.Background()
	namespace := "ai-conformance"

	_, _ = clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}, metav1.CreateOptions{})

	t.Run("NegativeIsolationTest", func(t *testing.T) {
		// Define a Pod that DOES NOT request GPU resources
		podName := "isolation-test-pod"
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: namespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    "prober",
						Image:   "ubuntu:22.04",
						Command: []string{"bash", "-c"},
						// Check for common accelerator device nodes (NVIDIA example)
						Args: []string{"ls /dev/nvidia* || echo 'No devices found'"},
					},
				},
				RestartPolicy: corev1.RestartPolicyNever,
				// Add tolerations if nodes are tainted for GPUs
				Tolerations: []corev1.Toleration{
					{
						Key:      "nvidia.com/gpu",
						Operator: corev1.TolerationOpExists,
						Effect:   corev1.TaintEffectNoSchedule,
					},
				},
			},
		}

		_, err := clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create Pod: %v", err)
		}
		defer clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})

		t.Log("Waiting for pod to complete probing...")
		for i := 0; i < 60; i++ {
			p, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if err == nil && (p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed) {
				break
			}
			time.Sleep(2 * time.Second)
		}

		// Verify Logs: Accelerator should NOT be visible
		podLogs, err := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{}).DoRaw(ctx)
		if err != nil {
			t.Fatalf("Failed to get pod logs: %v", err)
		}

		output := string(podLogs)
		t.Logf("Pod output: %s", output)

		// Isolation check: If we see device nodes, the test fails.
		if strings.Contains(output, "/dev/nvidia0") {
			t.Errorf("Security Violation: GPU device /dev/nvidia0 is visible inside a container that did not request it!")
		} else {
			t.Log("Verified: No accelerator devices found in unauthorized container.")
		}
	})
}
