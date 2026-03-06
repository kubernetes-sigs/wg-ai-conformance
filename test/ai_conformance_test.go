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
	resourcev1 "k8s.io/api/resource/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	t.Cleanup(func() {
		t.Logf("Cleaning up namespace %s...", namespace)
		err := clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Failed to cleanup namespace: %v", err)
		}
	})

	checkDRA(ctx, t, clientset)
	setupTestEnvironment(ctx, t, clientset, namespace)
	testResourceTemplateName := "gpu-claim-template"

	// Getting a GPU from inside a Pod that requests a GPU should succeed
	t.Run("PositiveAccessTest", func(t *testing.T) {
		podName := "pos-pod"
		claims := []corev1.PodResourceClaim{{
			Name:   "single-gpu",
			ResourceClaimTemplateName: &testResourceTemplateName,
		}}
		runPodWithClaim(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("prober")}, claims)
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "prober", true)
	})

	// Getting a GPU from inside a Pod that does not request a GPU should fail
	t.Run("NegativeIsolationTest", func(t *testing.T) {
		podName := "neg-pod"
		runPodWithClaim(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("prober")}, nil)
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "prober", false)
	})

	// Getting a GPU from another container inside a Pod should fail
	t.Run("MultiContainerIsolationTest", func(t *testing.T) {
		podName := "multi-container-pod"
		claims := []corev1.PodResourceClaim{{
			Name:   "single-gpu",
			ResourceClaimTemplateName: &testResourceTemplateName,
		}}
		runPodWithClaim(ctx, t, clientset, namespace, podName, []corev1.Container{acceleratorProbingContainer("authorized"), acceleratorProbingContainer("unauthorized")}, claims)

		// The first container can access the GPU, the second cannot
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "authorized", true)
		verifyHardwareInLogs(ctx, t, clientset, namespace, podName, "unauthorized", false)
	})
}

// Setup namespace and DRA templates
func setupTestEnvironment(ctx context.Context, t *testing.T, c *kubernetes.Clientset, ns string) {
	if _, err := c.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	template := &resourcev1.ResourceClaimTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu-claim-template", Namespace: ns},
		Spec: resourcev1.ResourceClaimTemplateSpec{
			Spec: resourcev1.ResourceClaimSpec{
				Devices: resourcev1.DeviceClaim{
					Requests: []resourcev1.DeviceRequest{{
						Name: "single-gpu",
						Exactly: &resourcev1.ExactDeviceRequest{
							DeviceClassName: "gpu.nvidia.com",
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
		t.Fatal("No ResourceSlices found! Ensure the DRA driver is running and nodes have available accelerators.")
	}

	for _, slice := range slices.Items {
		t.Logf("Found ResourceSlice: %s (Node: %s, Driver: %s)", slice.Name, *slice.Spec.NodeName, slice.Spec.Driver)
	}
}

// Runs a pod with the specified containers and claims
func runPodWithClaim(ctx context.Context, t *testing.T, c *kubernetes.Clientset, ns, name string, containers []corev1.Container, claims []corev1.PodResourceClaim) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.PodSpec{
			Containers: containers,
			Tolerations:    []corev1.Toleration{{Key: "nvidia.com/gpu", Operator: "Exists", Effect: "NoSchedule"}}, // for scheduling on accelerator nodes with taints
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
	for i := 0; i < 30; i++ {
		p, _ := c.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
		if p.Status.Phase == corev1.PodRunning { break }
		time.Sleep(2 * time.Second)
	}
}

// Verify that the container has access to the hardware (e.g. GPU) by checking its logs
func verifyHardwareInLogs(ctx context.Context, t *testing.T, c *kubernetes.Clientset, ns, podName, containerName string, expectSuccess bool) {
	var logs string
	found := false
	expectedText := "GPU"
	t.Logf("Waiting to see if Pod %s/%s logs contain '%s'...", podName, containerName, expectedText)
	for i := 0; i < 2; i++ {
		rawLogs, err := c.CoreV1().Pods(ns).GetLogs(podName, &corev1.PodLogOptions{Container: containerName}).DoRaw(ctx)
		if err == nil {
			logs = string(rawLogs)
			if strings.Contains(logs, expectedText) {
				found = true
				break
			}
		}
		time.Sleep(5 * time.Second)
	}

	if expectSuccess && !found {
		t.Errorf("FAIL: Container %s in Pod %s should see '%s'. Logs: %s", containerName, podName, expectedText, logs)
	} else if !expectSuccess && found {
		t.Errorf("VIOLATION: Unauthorized Container %s in Pod %s saw '%s'! Logs: %s", containerName, podName, expectedText, logs)
	} else {
		t.Logf("PASS: %s isolation/access verified.", containerName)
	}
}

// Returns a container that probes for GPUs
func acceleratorProbingContainer(name string) corev1.Container {
	return corev1.Container{
				Name:    name,
				Image:   "ubuntu:22.04",
				Command: []string{"bash", "-c"},
				Args: []string{"while [ 1 ]; do date; echo $(nvidia-smi -L || echo Waiting...); sleep 5; done"},
			}
}