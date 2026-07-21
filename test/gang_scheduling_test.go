package conformance

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// TestGangScheduling verifies the Gang Scheduling requirement.
// It creates standard Kubernetes Job resources with configurable labels/annotations
// to test all-or-nothing scheduling behavior without coupling to a specific implementation.
// Ref: https://github.com/kubernetes-sigs/ai-conformance/blob/main/kars/0053-gang-scheduling/README.md
func TestGangScheduling(t *testing.T) {
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
	namespace := *gangSchedulerNamespace
	cleanupNamespace := false

	if namespace == "" {
		namespace = randomNamespaceName("gang-scheduling")
		cleanupNamespace = true
		if _, err := clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			t.Fatalf("Failed to create namespace: %v", err)
		}
	}

	t.Cleanup(func() {
		if cleanupNamespace {
			t.Logf("Cleaning up namespace %s...", namespace)
			err := clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				t.Errorf("Failed to cleanup namespace: %v", err)
			}

			err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
				_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				return false, nil
			})
			if err != nil {
				t.Errorf("CLEANUP FAILURE: Failed to delete namespace %s: %v", namespace, err)
			}
		}
	})

	t.Run("PositiveGangScheduling", func(t *testing.T) {
		jobName := "pos-job"
		job := buildGenericGangSchedulingJob(namespace, jobName, 2, "100m", "128Mi")

		t.Cleanup(func() {
			deletePolicy := metav1.DeletePropagationBackground
			_ = clientset.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
		})

		t.Logf("Creating positive test job %s...", jobName)
		if _, err := clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{}); err != nil {
			t.Fatalf("Failed to create positive job: %v", err)
		}

		verifyJobComplete(ctx, t, clientset, namespace, jobName)
	})

	t.Run("NegativeGangScheduling", func(t *testing.T) {
		jobName := "neg-job"
		job := buildGenericGangSchedulingJob(namespace, jobName, 1000, "100m", "128Mi")

		t.Cleanup(func() {
			deletePolicy := metav1.DeletePropagationBackground
			_ = clientset.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
		})

		t.Logf("Creating negative test job %s...", jobName)
		if _, err := clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{}); err != nil {
			t.Fatalf("Failed to create negative job: %v", err)
		}

		verifyJobSuspendedOrPending(ctx, t, clientset, namespace, jobName)
	})
}

func buildGenericGangSchedulingJob(ns, name string, parallelism int, cpuReq, memReq string) *batchv1.Job {
	parallelism32 := int32(parallelism)
	
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    make(map[string]string),
		},
		Spec: batchv1.JobSpec{
			Parallelism: &parallelism32,
			Completions: &parallelism32,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"job-name": name},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "test-container",
							Image:   "busybox",
							Command: []string{"sh", "-c", "sleep 5"},
							Resources: corev1.ResourceRequirements{
								Requests: buildResourceList(cpuReq, memReq),
								Limits:   buildResourceList(cpuReq, memReq),
							},
						},
					},
				},
			},
		},
	}

	if *gangJobLabels != "" {
		pairs := strings.Split(*gangJobLabels, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				job.ObjectMeta.Labels[kv[0]] = kv[1]
			}
		}
	}

	return job
}

func verifyJobComplete(ctx context.Context, t *testing.T, clientset kubernetes.Interface, namespace, jobName string) {
	t.Logf("Waiting for positive job %s to complete...", jobName)
	err := wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		job, err := clientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
				return true, nil
			}
			if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
				t.Fatalf("Job %s failed instead of completing", jobName)
			}
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("Timeout waiting for positive job to complete: %v", err)
	}
	t.Logf("Positive job %s successfully completed", jobName)
}

func verifyJobSuspendedOrPending(ctx context.Context, t *testing.T, clientset kubernetes.Interface, namespace, jobName string) {
	verificationWindow := 10 * time.Second
	pollInterval := 1 * time.Second
	t.Logf("Verifying negative job %s remains suspended or pending without partial pods for %v...", jobName, verificationWindow)

	err := wait.PollUntilContextTimeout(ctx, pollInterval, verificationWindow, true, func(ctx context.Context) (bool, error) {
		job, err := clientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if job.Spec.Suspend != nil && *job.Spec.Suspend {
			// Job is suspended, which is valid for gang scheduling (e.g., Kueue)
			// Continue polling to ensure it remains suspended
			return false, nil
		}

		// If not suspended (e.g., Volcano), verify no pods are running or bound to a Node
		pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "job-name=" + jobName,
		})
		if err != nil {
			return false, err
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning || pod.Spec.NodeName != "" {
				return false, fmt.Errorf("negative job %s has a running or bound pod %s (partial scheduling)", jobName, pod.Name)
			}
		}

		return false, nil
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "timed out") || strings.Contains(err.Error(), "context deadline exceeded") {
			t.Logf("Negative job %s successfully remained suspended or pending with 0 bound pods throughout the verification window.", jobName)
		} else {
			t.Fatalf("Failed to verify negative job: %v", err)
		}
	} else {
		t.Fatalf("Unexpected success from polling loop")
	}
}

func buildResourceList(cpuReq, memReq string) corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse(cpuReq),
		corev1.ResourceMemory: resource.MustParse(memReq),
	}
}
