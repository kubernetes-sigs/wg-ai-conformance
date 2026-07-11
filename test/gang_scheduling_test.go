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

	kueuev1beta1 "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	kueueclientset "sigs.k8s.io/kueue/client-go/clientset/versioned"
)

// TestGangScheduling verifies the Gang Scheduling requirement using Kueue.
// Ref: https://github.com/cncf/k8s-ai-conformance/blob/main/kars/0053-gang-scheduling/README.md
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

	kueueClient, err := kueueclientset.NewForConfig(config)
	if err != nil {
		t.Fatalf("Error creating kueue client: %v", err)
	}

	ctx := context.Background()
	namespace := randomNamespaceName("gang-scheduling")
	flavorName := randomNamespaceName("test-flavor")
	cqName := randomNamespaceName("test-cq")
	lqName := randomNamespaceName("test-lq")

	t.Cleanup(func() {
		t.Logf("Cleaning up namespace %s and Kueue objects...", namespace)

		err := clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("Failed to cleanup namespace: %v", err)
		}

		err = kueueClient.KueueV1beta1().ClusterQueues().Delete(ctx, cqName, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("Failed to cleanup ClusterQueue: %v", err)
		}

		err = kueueClient.KueueV1beta1().ResourceFlavors().Delete(ctx, flavorName, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("Failed to cleanup ResourceFlavor: %v", err)
		}

		err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
			_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			t.Errorf("CLEANUP FAILURE: Failed to delete namespace %s: %v.", namespace, err)
		}
		
		err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
			_, err := kueueClient.KueueV1beta1().ClusterQueues().Get(ctx, cqName, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			t.Errorf("CLEANUP FAILURE: Failed to delete ClusterQueue %s: %v.", cqName, err)
		}

		err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
			_, err := kueueClient.KueueV1beta1().ResourceFlavors().Get(ctx, flavorName, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			t.Errorf("CLEANUP FAILURE: Failed to delete ResourceFlavor %s: %v.", flavorName, err)
		}
	})

	if _, err := clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	rf := buildResourceFlavor(flavorName)
	if _, err := kueueClient.KueueV1beta1().ResourceFlavors().Create(ctx, rf, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create ResourceFlavor: %v", err)
	}

	cq := buildClusterQueue(cqName, flavorName, "1", "1Gi")
	if _, err := kueueClient.KueueV1beta1().ClusterQueues().Create(ctx, cq, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create ClusterQueue: %v", err)
	}

	lq := buildLocalQueue(namespace, lqName, cqName)
	if _, err := kueueClient.KueueV1beta1().LocalQueues(namespace).Create(ctx, lq, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create LocalQueue: %v", err)
	}

	t.Logf("Waiting for ClusterQueue %s to become Active...", cqName)
	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		gotCq, err := kueueClient.KueueV1beta1().ClusterQueues().Get(ctx, cqName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range gotCq.Status.Conditions {
			if cond.Type == kueuev1beta1.ClusterQueueActive && cond.Status == metav1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("ClusterQueue %s did not become Active: %v", cqName, err)
	}

	t.Run("PositiveGangScheduling", func(t *testing.T) {
		jobName := "pos-job"
		job := buildGangSchedulingJob(namespace, jobName, lqName, 3, "100m", "64Mi")
		
		t.Cleanup(func() {
			deletePolicy := metav1.DeletePropagationBackground
			_ = clientset.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
		})

		t.Logf("Creating positive test job %s...", jobName)
		if _, err := clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{}); err != nil {
			t.Fatalf("Failed to create positive job: %v", err)
		}

		t.Logf("Waiting for job %s to be unsuspended and complete...", jobName)
		err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
			p, err := clientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			
			// We want to see it either Succeeded or at least unsuspended and running
			if p.Status.Succeeded == 3 {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			t.Fatalf("Job %s did not complete successfully within 2m. Error: %v", jobName, err)
		}
		t.Logf("PASS: Positive gang scheduling verified.")
	})

	t.Run("NegativeGangScheduling", func(t *testing.T) {
		jobName := "neg-job"
		job := buildGangSchedulingJob(namespace, jobName, lqName, 3, "500m", "512Mi")
		
		t.Cleanup(func() {
			deletePolicy := metav1.DeletePropagationBackground
			_ = clientset.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
		})

		t.Logf("Creating negative test job %s...", jobName)
		if _, err := clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{}); err != nil {
			t.Fatalf("Failed to create negative job: %v", err)
		}

		t.Logf("Waiting 30s to ensure job %s remains suspended and no pods are scheduled...", jobName)
		
		time.Sleep(30 * time.Second)

		p, err := clientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Failed to get negative job: %v", err)
		}

		if p.Spec.Suspend == nil || !*p.Spec.Suspend {
			t.Errorf("FAIL: Expected job to remain suspended, but Suspend is false or nil")
		}

		if p.Status.Active > 0 || p.Status.Succeeded > 0 || p.Status.Failed > 0 {
			t.Errorf("FAIL: Expected 0 active/succeeded/failed pods, got Active=%d, Succeeded=%d, Failed=%d", p.Status.Active, p.Status.Succeeded, p.Status.Failed)
		} else {
			t.Logf("PASS: Negative gang scheduling verified (all-or-nothing behavior enforced).")
		}
	})

	t.Run("Cleanup", func(t *testing.T) {
		t.Logf("Verifying cleanup on ClusterQueue %s...", cqName)
		err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
			gotCq, err := kueueClient.KueueV1beta1().ClusterQueues().Get(ctx, cqName, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			if gotCq.Status.AdmittedWorkloads == 0 && gotCq.Status.PendingWorkloads == 0 && gotCq.Status.ReservingWorkloads == 0 {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			t.Fatalf("ClusterQueue %s did not return to 0 workloads: %v", cqName, err)
		}
		t.Logf("PASS: Cleanup verified.")
	})
}
