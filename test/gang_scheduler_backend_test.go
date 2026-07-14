package conformance

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"os/exec"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	kueuev1beta1 "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	kueueclientset "sigs.k8s.io/kueue/client-go/clientset/versioned"
)

type gangSchedulerBackend interface {
	setup(ctx context.Context, t *testing.T, clientset kubernetes.Interface, restConfig *rest.Config, namespace string)

	teardown(ctx context.Context, t *testing.T)

	buildPositiveJob(namespace, name string) *batchv1.Job

	buildNegativeJob(namespace, name string) *batchv1.Job

	verifyAdmittedAndComplete(ctx context.Context, t *testing.T, clientset kubernetes.Interface, namespace, jobName string)

	verifyNotAdmitted(ctx context.Context, t *testing.T, clientset kubernetes.Interface, namespace, jobName string)

	verifyResourcesReleased(ctx context.Context, t *testing.T)
}

var supportedGangSchedulers = []string{"kueue"}

func lookupGangScheduler(name string) (gangSchedulerBackend, error) {
	switch name {
	case "kueue":
		return &kueueBackend{}, nil
	default:
		return nil, fmt.Errorf("unsupported gang scheduler %q; supported: %s",
			name, strings.Join(supportedGangSchedulers, ", "))
	}
}

type kueueBackend struct {
	kueueClient kueueclientset.Interface
	flavorName  string
	cqName      string
	lqName      string
}

func (k *kueueBackend) setup(ctx context.Context, t *testing.T, clientset kubernetes.Interface, restConfig *rest.Config, namespace string) {
	t.Logf("Installing Kueue...")
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "--server-side", "-f", "https://github.com/kubernetes-sigs/kueue/releases/download/v0.18.2/manifests.yaml")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to install Kueue: %v\nOutput: %s", err, string(out))
	}

	t.Logf("Waiting for Kueue controller manager to be ready...")
	err := wait.PollUntilContextTimeout(ctx, 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		deployment, err := clientset.AppsV1().Deployments("kueue-system").Get(ctx, "kueue-controller-manager", metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		for _, cond := range deployment.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("Kueue controller manager did not become ready: %v", err)
	}
	t.Logf("Kueue installed successfully.")

	k.kueueClient, err = kueueclientset.NewForConfig(restConfig)
	if err != nil {
		t.Fatalf("Error creating Kueue client: %v", err)
	}

	k.flavorName = randomNamespaceName("test-flavor")
	k.cqName = randomNamespaceName("test-cq")
	k.lqName = randomNamespaceName("test-lq")

	rf := buildResourceFlavor(k.flavorName)
	if _, err := k.kueueClient.KueueV1beta1().ResourceFlavors().Create(ctx, rf, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create ResourceFlavor (is Kueue installed?): %v", err)
	}

	cq := buildClusterQueue(k.cqName, k.flavorName, "1", "1Gi")
	if _, err := k.kueueClient.KueueV1beta1().ClusterQueues().Create(ctx, cq, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create ClusterQueue: %v", err)
	}

	lq := buildLocalQueue(namespace, k.lqName, k.cqName)
	if _, err := k.kueueClient.KueueV1beta1().LocalQueues(namespace).Create(ctx, lq, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create LocalQueue: %v", err)
	}

	t.Logf("Waiting for ClusterQueue %s to become Active...", k.cqName)
	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		gotCq, err := k.kueueClient.KueueV1beta1().ClusterQueues().Get(ctx, k.cqName, metav1.GetOptions{})
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
		t.Fatalf("ClusterQueue %s did not become Active: %v", k.cqName, err)
	}
}

func (k *kueueBackend) teardown(ctx context.Context, t *testing.T) {
	if k.kueueClient == nil {
		return
	}

	t.Logf("Cleaning up Kueue objects (ClusterQueue: %s, ResourceFlavor: %s)...", k.cqName, k.flavorName)

	err := k.kueueClient.KueueV1beta1().ClusterQueues().Delete(ctx, k.cqName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		t.Errorf("Failed to cleanup ClusterQueue: %v", err)
	}

	err = k.kueueClient.KueueV1beta1().ResourceFlavors().Delete(ctx, k.flavorName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		t.Errorf("Failed to cleanup ResourceFlavor: %v", err)
	}

	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		_, err := k.kueueClient.KueueV1beta1().ClusterQueues().Get(ctx, k.cqName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Errorf("CLEANUP FAILURE: Failed to delete ClusterQueue %s: %v.", k.cqName, err)
	}

	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		_, err := k.kueueClient.KueueV1beta1().ResourceFlavors().Get(ctx, k.flavorName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Errorf("CLEANUP FAILURE: Failed to delete ResourceFlavor %s: %v.", k.flavorName, err)
	}
}

func (k *kueueBackend) buildPositiveJob(namespace, name string) *batchv1.Job {
	return buildGangSchedulingJob(namespace, name, k.lqName, 3, "100m", "64Mi")
}

func (k *kueueBackend) buildNegativeJob(namespace, name string) *batchv1.Job {
	return buildGangSchedulingJob(namespace, name, k.lqName, 3, "500m", "512Mi")
}

func (k *kueueBackend) verifyAdmittedAndComplete(ctx context.Context, t *testing.T, clientset kubernetes.Interface, namespace, jobName string) {
	t.Logf("Waiting for job %s to be unsuspended and complete...", jobName)
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		p, err := clientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if p.Status.Succeeded == 3 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("Job %s did not complete successfully within 2m. Error: %v", jobName, err)
	}
	t.Logf("PASS: Positive gang scheduling verified.")
}

func (k *kueueBackend) verifyNotAdmitted(ctx context.Context, t *testing.T, clientset kubernetes.Interface, namespace, jobName string) {
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
		t.Errorf("FAIL: Expected 0 active/succeeded/failed pods, got Active=%d, Succeeded=%d, Failed=%d",
			p.Status.Active, p.Status.Succeeded, p.Status.Failed)
	} else {
		t.Logf("PASS: Negative gang scheduling verified (all-or-nothing behavior enforced).")
	}
}

func (k *kueueBackend) verifyResourcesReleased(ctx context.Context, t *testing.T) {
	t.Logf("Verifying cleanup on ClusterQueue %s...", k.cqName)
	err := wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		gotCq, err := k.kueueClient.KueueV1beta1().ClusterQueues().Get(ctx, k.cqName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if gotCq.Status.AdmittedWorkloads == 0 && gotCq.Status.PendingWorkloads == 0 && gotCq.Status.ReservingWorkloads == 0 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("ClusterQueue %s did not return to 0 workloads: %v", k.cqName, err)
	}
	t.Logf("PASS: Cleanup verified.")
}
