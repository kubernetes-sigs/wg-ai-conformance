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

// TestGangScheduling verifies the Gang Scheduling requirement.
// It uses a pluggable backend (selected via -gang-scheduler flag) to
// test all-or-nothing scheduling behavior without coupling to a specific
// gang scheduling implementation.
// Ref: https://github.com/kubernetes-sigs/ai-conformance/blob/main/kars/0053-gang-scheduling/README.md
func TestGangScheduling(t *testing.T) {
	if !flag.Parsed() {
		flag.Parse()
	}

	backend, err := lookupGangScheduler(*gangScheduler)
	if err != nil {
		t.Fatalf("Invalid -gang-scheduler: %v", err)
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
	namespace := randomNamespaceName("gang-scheduling")

	t.Cleanup(func() {
		t.Logf("Cleaning up namespace %s and gang scheduler objects...", namespace)

		err := clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("Failed to cleanup namespace: %v", err)
		}

		backend.teardown(ctx, t)

		err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
			_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			t.Errorf("CLEANUP FAILURE: Failed to delete namespace %s: %v. "+
				"Please ensure this namespace is terminated manually to avoid resource leaks. "+
				"Rerunning the tests might fail if the namespace is not deleted.",
				namespace, err)
		}
	})

	if _, err := clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	backend.setup(ctx, t, clientset, config, namespace)

	t.Run("PositiveGangScheduling", func(t *testing.T) {
		jobName := "pos-job"
		job := backend.buildPositiveJob(namespace, jobName)

		t.Cleanup(func() {
			deletePolicy := metav1.DeletePropagationBackground
			_ = clientset.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
		})

		t.Logf("Creating positive test job %s...", jobName)
		if _, err := clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{}); err != nil {
			t.Fatalf("Failed to create positive job: %v", err)
		}

		backend.verifyAdmittedAndComplete(ctx, t, clientset, namespace, jobName)
	})

	t.Run("NegativeGangScheduling", func(t *testing.T) {
		jobName := "neg-job"
		job := backend.buildNegativeJob(namespace, jobName)

		t.Cleanup(func() {
			deletePolicy := metav1.DeletePropagationBackground
			_ = clientset.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
		})

		t.Logf("Creating negative test job %s...", jobName)
		if _, err := clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{}); err != nil {
			t.Fatalf("Failed to create negative job: %v", err)
		}

		backend.verifyNotAdmitted(ctx, t, clientset, namespace, jobName)
	})

	t.Run("Cleanup", func(t *testing.T) {
		backend.verifyResourcesReleased(ctx, t)
	})
}
