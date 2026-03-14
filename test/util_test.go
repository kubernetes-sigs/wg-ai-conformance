package conformance

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type AcceleratorConfig struct {
	DeviceClass string
	TaintKey    string
	DevicePath  string
}

var (
	kubeconfig         *string
	acceleratorType    *string
	acceleratorConfigs = map[string]AcceleratorConfig{
		"nvidia": {
			DeviceClass: "gpu.nvidia.com",
			TaintKey:    "nvidia.com/gpu",
			DevicePath:  "/dev/nvidia*",
		},
		// Add other vendors here
	}
	testResourceTemplateName = "accelerator-claim-template"
	testRequestName          = "single-accelerator"
)

func init() {
	kubeconfig = flag.String(clientcmd.RecommendedConfigPathFlag, "", "absolute path to the kubeconfig file")
	acceleratorType = flag.String("accelerator-type", "nvidia", "The type of accelerator to test. Supported types: 'nvidia' (default). Support for other types is being added.")
}

// Setup namespace and DRA templates
func setupTestEnvironment(ctx context.Context, t *testing.T, c *kubernetes.Clientset, ns string) {
	if _, err := c.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	template := &resourcev1.ResourceClaimTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: testResourceTemplateName, Namespace: ns},
		Spec: resourcev1.ResourceClaimTemplateSpec{
			Spec: resourcev1.ResourceClaimSpec{
				Devices: resourcev1.DeviceClaim{
					Requests: []resourcev1.DeviceRequest{{
						Name: testRequestName,
						Exactly: &resourcev1.ExactDeviceRequest{
							DeviceClassName: acceleratorConfigs[*acceleratorType].DeviceClass,
							Count:           1,
						},
					}},
				},
			},
		},
	}

	if _, err := c.ResourceV1().ResourceClaimTemplates(ns).Create(ctx, template, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("Failed to create resource claim template: %v", err)
	}
}

// Verifies that DRA is available and functional
func checkDRA(ctx context.Context, t *testing.T, c *kubernetes.Clientset) {
	t.Log("Checking for available ResourceSlices...")

	slices, err := c.ResourceV1().ResourceSlices().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list ResourceSlices: %v. Is the DRA API enabled?", err)
	}

	if len(slices.Items) == 0 {
		t.Fatal("ENVIRONMENT ERROR: No ResourceSlices found! Ensure the DRA driver is running and nodes have available accelerators.")
	}

	for _, slice := range slices.Items {
		t.Logf("Checking environment: Found ResourceSlice: %s (Node: %s, Driver: %s)", slice.Name, *slice.Spec.NodeName, slice.Spec.Driver)
	}
}

// Runs a pod with the specified containers and claims
func runPodWithClaim(ctx context.Context, t *testing.T, c *kubernetes.Clientset, ns, name string, containers []corev1.Container, claims []corev1.PodResourceClaim) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.PodSpec{
			Containers:  containers,
			Tolerations: []corev1.Toleration{{Key: acceleratorConfigs[*acceleratorType].TaintKey, Operator: "Exists", Effect: "NoSchedule"}}, // for scheduling on accelerator nodes with taints
		},
	}

	// Only the first container is granted the claim
	if len(claims) > 0 {
		pod.Spec.Containers[0].Resources.Claims = []corev1.ResourceClaim{{Name: claims[0].Name}}
		pod.Spec.ResourceClaims = claims
	}

	_, err := c.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create Pod: %v", err)
	}

	t.Logf("Waiting for Pod %s to be running...", name)
	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		p, err := c.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return p.Status.Phase == corev1.PodRunning, nil
	})

	if err != nil {
		p, _ := c.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
		t.Fatalf("Pod %s failed to reach Running phase within 1m. Current phase: %s, Error: %v", name, p.Status.Phase, err)
	}
}

// Verify that the container has access to the hardware (e.g. GPU) by checking its logs
func verifyHardwareInLogs(ctx context.Context, t *testing.T, c *kubernetes.Clientset, ns, podName, containerName string, expectSuccess bool) {
	var logs string
	pass := false
	var expectedText string
	if expectSuccess {
		expectedText = "RESULT: ACCELERATOR_FOUND"
	} else {
		expectedText = "RESULT: ACCELERATOR_MISSING"
	}
	t.Logf("Waiting to see if Pod %s/%s logs contain '%s'...", podName, containerName, expectedText)
	for i := 0; i < 2; i++ {
		rawLogs, err := c.CoreV1().Pods(ns).GetLogs(podName, &corev1.PodLogOptions{Container: containerName}).DoRaw(ctx)
		if err == nil {
			logs = string(rawLogs)
			if strings.Contains(logs, expectedText) {
				pass = true
				break
			}
		}
		time.Sleep(5 * time.Second)
	}

	if pass {
		t.Logf("PASS: %s isolation/access verified.", containerName)
	} else if expectSuccess {
		t.Errorf("FAIL: Container %s in Pod %s should see '%s'. Logs: %s", containerName, podName, expectedText, logs)
	} else {
		t.Errorf("VIOLATION: Unauthorized Container %s in Pod %s saw '%s'! Logs: %s", containerName, podName, expectedText, logs)
	}
}

// Returns a container that probes for accelerator
func acceleratorProbingContainer(name string) corev1.Container {
	return corev1.Container{
		Name:    name,
		Image:   "ubuntu:22.04",
		Command: []string{"/bin/sh", "-c"},
		Args: []string{
			fmt.Sprintf("if ls %s > /dev/null 2>&1; then echo 'RESULT: ACCELERATOR_FOUND'; else echo 'RESULT: ACCELERATOR_MISSING'; fi; sleep 3600", acceleratorConfigs[*acceleratorType].DevicePath),
		},
	}
}

func randomNamespaceName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, rand.String(5))
}
