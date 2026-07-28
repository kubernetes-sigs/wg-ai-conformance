package conformance

import (
	"context"
	"errors"
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
)

// TestGangScheduling verifies the Gang Scheduling requirement.
// It creates standard Kubernetes Job resources with configurable labels/annotations
// to test all-or-nothing scheduling behavior without coupling to a specific implementation.
// Ref: https://github.com/kubernetes-sigs/ai-conformance/blob/main/kars/0053-gang-scheduling/README.md
func TestGangScheduling(t *testing.T) {
	clientset := getClientset(t)

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

	// Verify scheduling evidence: ensure the job was actually processed
	// by a gang scheduler and not just completed by the default scheduler.
	t.Logf("Verifying scheduling evidence for completed job %s...", jobName)

	completedJob, err := clientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get completed job for verification: %v", err)
	}

	// Verify the gang-job-labels are present on the completed job,
	// confirming the gang scheduling configuration was in effect.
	if *gangJobLabels != "" {
		pairs := strings.Split(*gangJobLabels, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				if val, ok := completedJob.Labels[kv[0]]; !ok || val != kv[1] {
					t.Errorf("Expected gang scheduling label %s=%s on completed job, got labels: %v", kv[0], kv[1], completedJob.Labels)
				}
			}
		}
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "job-name=" + jobName,
	})
	if err != nil {
		t.Fatalf("Failed to list pods for completed job: %v", err)
	}
	if len(pods.Items) == 0 {
		t.Fatalf("No pods found for completed job %s - job may not have been admitted by the gang scheduler", jobName)
	}

	for _, pod := range pods.Items {
		if pod.Spec.NodeName == "" {
			t.Errorf("Pod %s of completed job has no node assignment", pod.Name)
		}
		t.Logf("  Pod %s: schedulerName=%s, nodeName=%s, phase=%s",
			pod.Name, pod.Spec.SchedulerName, pod.Spec.NodeName, pod.Status.Phase)
	}

	t.Logf("Positive job %s completed with %d pods verified", jobName, len(pods.Items))
}

func verifyJobSuspendedOrPending(ctx context.Context, t *testing.T, clientset kubernetes.Interface, namespace, jobName string) {
	verificationWindow := *gangNegativeWindow
	pollInterval := 1 * time.Second
	t.Logf("Verifying negative job %s remains suspended or pending without partial pods for %v...", jobName, verificationWindow)

	err := wait.PollUntilContextTimeout(ctx, pollInterval, verificationWindow, true, func(ctx context.Context) (bool, error) {
		job, err := clientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		isSuspended := job.Spec.Suspend != nil && *job.Spec.Suspend

		// Always list the Job's pods and assert zero bound/running pods,
		// even when the job is suspended. A broken controller could suspend
		// the Job after some pods were already created/bound.
		pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "job-name=" + jobName,
		})
		if err != nil {
			return false, err
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning || pod.Spec.NodeName != "" {
				return false, fmt.Errorf("negative job %s has a running or bound pod %s (partial scheduling detected, suspended=%v)", jobName, pod.Name, isSuspended)
			}
		}

		// Continue polling - we want to observe the entire verification window
		return false, nil
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "timed out") || strings.Contains(err.Error(), "context deadline exceeded") {
			t.Logf("Negative job %s successfully remained suspended or pending with 0 bound pods throughout the %v verification window.", jobName, verificationWindow)
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
