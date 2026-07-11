package conformance

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	corev1 "k8s.io/api/core/v1"
)

func TestBuildGangSchedulingJob(t *testing.T) {
	ns := "test-ns"
	name := "test-job"
	queueName := "test-queue"
	parallelism := 3
	cpuReq := "100m"
	memReq := "64Mi"

	job := buildGangSchedulingJob(ns, name, queueName, parallelism, cpuReq, memReq)

	if job.Namespace != ns {
		t.Errorf("expected namespace %s, got %s", ns, job.Namespace)
	}
	if job.Name != name {
		t.Errorf("expected name %s, got %s", name, job.Name)
	}

	labels := job.ObjectMeta.Labels
	if labels == nil || labels["kueue.x-k8s.io/queue-name"] != queueName {
		t.Errorf("expected queue-name label %s, got %v", queueName, labels["kueue.x-k8s.io/queue-name"])
	}

	if job.Spec.Parallelism == nil || *job.Spec.Parallelism != int32(parallelism) {
		t.Errorf("expected parallelism %d, got %v", parallelism, job.Spec.Parallelism)
	}
	if job.Spec.Completions == nil || *job.Spec.Completions != int32(parallelism) {
		t.Errorf("expected completions %d, got %v", parallelism, job.Spec.Completions)
	}
	if job.Spec.Suspend == nil || !*job.Spec.Suspend {
		t.Errorf("expected suspend true, got %v", job.Spec.Suspend)
	}

	if len(job.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(job.Spec.Template.Spec.Containers))
	}

	reqs := job.Spec.Template.Spec.Containers[0].Resources.Requests
	if reqs[corev1.ResourceCPU] != resource.MustParse(cpuReq) {
		t.Errorf("expected CPU request %s, got %v", cpuReq, reqs[corev1.ResourceCPU])
	}
	if reqs[corev1.ResourceMemory] != resource.MustParse(memReq) {
		t.Errorf("expected Memory request %s, got %v", memReq, reqs[corev1.ResourceMemory])
	}
}

func TestBuildKueueObjects(t *testing.T) {
	t.Run("ResourceFlavor", func(t *testing.T) {
		rf := buildResourceFlavor("test-flavor")
		if rf.Name != "test-flavor" {
			t.Errorf("expected name test-flavor, got %s", rf.Name)
		}
	})

	t.Run("ClusterQueue", func(t *testing.T) {
		cq := buildClusterQueue("test-cq", "test-flavor", "1", "1Gi")
		if cq.Name != "test-cq" {
			t.Errorf("expected name test-cq, got %s", cq.Name)
		}
		if len(cq.Spec.ResourceGroups) != 1 {
			t.Fatalf("expected 1 resource group, got %d", len(cq.Spec.ResourceGroups))
		}
		rg := cq.Spec.ResourceGroups[0]
		if len(rg.Flavors) != 1 {
			t.Fatalf("expected 1 flavor in resource group, got %d", len(rg.Flavors))
		}
		flavor := rg.Flavors[0]
		if string(flavor.Name) != "test-flavor" {
			t.Errorf("expected flavor name test-flavor, got %s", flavor.Name)
		}
		if len(flavor.Resources) != 2 {
			t.Fatalf("expected 2 resources, got %d", len(flavor.Resources))
		}
		
		var cpuRes, memRes bool
		for _, res := range flavor.Resources {
			if res.Name == corev1.ResourceCPU {
				cpuRes = true
				if res.NominalQuota != resource.MustParse("1") {
					t.Errorf("expected CPU quota 1, got %v", res.NominalQuota)
				}
			} else if res.Name == corev1.ResourceMemory {
				memRes = true
				if res.NominalQuota != resource.MustParse("1Gi") {
					t.Errorf("expected Memory quota 1Gi, got %v", res.NominalQuota)
				}
			}
		}
		if !cpuRes || !memRes {
			t.Errorf("missing CPU or Memory resources")
		}
	})

	t.Run("LocalQueue", func(t *testing.T) {
		lq := buildLocalQueue("test-ns", "test-lq", "test-cq")
		if lq.Name != "test-lq" {
			t.Errorf("expected name test-lq, got %s", lq.Name)
		}
		if lq.Namespace != "test-ns" {
			t.Errorf("expected namespace test-ns, got %s", lq.Namespace)
		}
		if string(lq.Spec.ClusterQueue) != "test-cq" {
			t.Errorf("expected cluster queue test-cq, got %s", lq.Spec.ClusterQueue)
		}
	})
}
