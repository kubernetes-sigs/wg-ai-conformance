package conformance

import (
	"context"
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// Unit tests for allocation-mode detection and test-pod construction. These
// run against a fake clientset and need no cluster, unlike
// TestSecureAcceleratorAccess.

func nvidiaConfig(t *testing.T) AcceleratorConfig {
	t.Helper()
	cfg, err := lookupAcceleratorConfig("nvidia")
	if err != nil {
		t.Fatalf("nvidia config must exist: %v", err)
	}
	return cfg
}

func gpuDeviceClass(name string) *resourcev1.DeviceClass {
	return &resourcev1.DeviceClass{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

func extendedResourceDeviceClass(name, extendedResourceName string) *resourcev1.DeviceClass {
	return &resourcev1.DeviceClass{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       resourcev1.DeviceClassSpec{ExtendedResourceName: &extendedResourceName},
	}
}

// gpuDevices prefixes device names with the slice name: device names must be
// unique across an entire pool, and a multi-slice pool with duplicates is
// invalid to the scheduler's allocator.
func gpuDevices(sliceName string, deviceCount int) []resourcev1.Device {
	devices := make([]resourcev1.Device, 0, deviceCount)
	for i := 0; i < deviceCount; i++ {
		devices = append(devices, resourcev1.Device{Name: sliceName + "-gpu-" + string(rune('0'+i))})
	}
	return devices
}

func gpuResourceSlice(name, driver, nodeName string, deviceCount int) *resourcev1.ResourceSlice {
	return &resourcev1.ResourceSlice{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: resourcev1.ResourceSliceSpec{
			Driver:   driver,
			NodeName: &nodeName,
			Pool:     resourcev1.ResourcePool{Name: name, Generation: 1, ResourceSliceCount: 1},
			Devices:  gpuDevices(name, deviceCount),
		},
	}
}

func gpuLabelSelector(key, value string) *corev1.NodeSelector {
	return &corev1.NodeSelector{
		NodeSelectorTerms: []corev1.NodeSelectorTerm{{
			MatchExpressions: []corev1.NodeSelectorRequirement{{
				Key: key, Operator: corev1.NodeSelectorOpIn, Values: []string{value},
			}},
		}},
	}
}

func selectorResourceSlice(name, driver string, selector *corev1.NodeSelector, deviceCount int) *resourcev1.ResourceSlice {
	return &resourcev1.ResourceSlice{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: resourcev1.ResourceSliceSpec{
			Driver:       driver,
			NodeSelector: selector,
			Pool:         resourcev1.ResourcePool{Name: name, Generation: 1, ResourceSliceCount: 1},
			Devices:      gpuDevices(name, deviceCount),
		},
	}
}

func allNodesResourceSlice(name, driver string, deviceCount int) *resourcev1.ResourceSlice {
	allNodes := true
	return &resourcev1.ResourceSlice{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: resourcev1.ResourceSliceSpec{
			Driver:   driver,
			AllNodes: &allNodes,
			Pool:     resourcev1.ResourcePool{Name: name, Generation: 1, ResourceSliceCount: 1},
			Devices:  gpuDevices(name, deviceCount),
		},
	}
}

func pooledResourceSlice(name, driver, nodeName, poolName string, generation int64, deviceCount int, sliceCount int64) *resourcev1.ResourceSlice {
	slice := gpuResourceSlice(name, driver, nodeName, deviceCount)
	slice.Spec.Pool = resourcev1.ResourcePool{Name: poolName, Generation: generation, ResourceSliceCount: sliceCount}
	return slice
}

func taintedResourceSlice(name, driver, nodeName string) *resourcev1.ResourceSlice {
	slice := gpuResourceSlice(name, driver, nodeName, 1)
	slice.Spec.Devices[0].Taints = []resourcev1.DeviceTaint{{
		Key: "example.com/unhealthy", Effect: resourcev1.DeviceTaintEffectNoSchedule,
	}}
	return slice
}

func perDeviceResourceSlice(name, driver, deviceNodeName string) *resourcev1.ResourceSlice {
	perDevice := true
	return &resourcev1.ResourceSlice{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: resourcev1.ResourceSliceSpec{
			Driver:                 driver,
			PerDeviceNodeSelection: &perDevice,
			Pool:                   resourcev1.ResourcePool{Name: name, Generation: 1, ResourceSliceCount: 1},
			Devices:                []resourcev1.Device{{Name: "gpu-0", NodeName: &deviceNodeName}},
		},
	}
}

func labeledNode(name string, labels map[string]string, ready bool) *corev1.Node {
	node := gpuNode(name, 0, ready, false)
	node.Labels = labels
	return node
}

func gpuNode(name string, gpus int64, ready, unschedulable bool) *corev1.Node {
	readyStatus := corev1.ConditionFalse
	if ready {
		readyStatus = corev1.ConditionTrue
	}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       corev1.NodeSpec{Unschedulable: unschedulable},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: readyStatus}},
		},
	}
	if gpus > 0 {
		node.Status.Allocatable = corev1.ResourceList{
			corev1.ResourceName("nvidia.com/gpu"): *resource.NewQuantity(gpus, resource.DecimalSI),
		}
	}
	return node
}

func TestDetectAllocationMode(t *testing.T) {
	cfg := nvidiaConfig(t)

	tests := []struct {
		name      string
		requested string
		objects   []runtime.Object
		wantMode  string
		wantNode  string
		wantErr   string
	}{
		{
			name:      "auto prefers DRA when DeviceClass and node-backed slices exist",
			requested: allocationModeAuto,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				gpuResourceSlice("node1-gpu", "gpu.nvidia.com", "node1", 8),
				gpuNode("node1", 8, true, false),
			},
			wantMode: allocationModeDRA,
			wantNode: "node1",
		},
		{
			name:      "auto falls back to device plugin on stale DeviceClass without slices",
			requested: allocationModeAuto,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				gpuNode("node1", 8, true, false),
			},
			wantMode: allocationModeDevicePlugin,
			wantNode: "node1",
		},
		{
			name:      "auto falls back when slices have no devices",
			requested: allocationModeAuto,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				gpuResourceSlice("node1-gpu", "gpu.nvidia.com", "node1", 0),
				gpuNode("node1", 8, true, false),
			},
			wantMode: allocationModeDevicePlugin,
			wantNode: "node1",
		},
		{
			name:      "auto falls back when slices belong to an unrelated driver",
			requested: allocationModeAuto,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				gpuResourceSlice("node1-cd", "compute-domain.nvidia.com", "node1", 2),
				gpuNode("node1", 8, true, false),
			},
			wantMode: allocationModeDevicePlugin,
			wantNode: "node1",
		},
		{
			// The motivating managed-cluster shape: the NVIDIA DRA driver
			// runs ComputeDomain-only (no gpu.nvidia.com DeviceClass or GPU
			// slices at all), whole GPUs come from the device plugin.
			name:      "auto falls back on a ComputeDomain-only cluster without a GPU DeviceClass",
			requested: allocationModeAuto,
			objects: []runtime.Object{
				gpuDeviceClass("compute-domain-default-channel.nvidia.com"),
				gpuResourceSlice("node1-cd", "compute-domain.nvidia.com", "node1", 2),
				gpuNode("node1", 8, true, false),
			},
			wantMode: allocationModeDevicePlugin,
			wantNode: "node1",
		},
		{
			name:      "auto falls back when the only slice-backed node is NotReady",
			requested: allocationModeAuto,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				gpuResourceSlice("down-gpu", "gpu.nvidia.com", "down-node", 8),
				gpuNode("down-node", 0, false, false),
				gpuNode("node1", 8, true, false),
			},
			wantMode: allocationModeDevicePlugin,
			wantNode: "node1",
		},
		{
			name:      "auto errors when neither mechanism is usable",
			requested: allocationModeAuto,
			objects:   []runtime.Object{gpuNode("cpu-node", 0, true, false)},
			wantErr:   "no usable accelerator allocation mechanism",
		},
		{
			name:      "explicit dra succeeds when a Ready node backs the slices",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				gpuResourceSlice("node1-gpu", "gpu.nvidia.com", "node1", 4),
				gpuNode("node1", 0, true, false),
			},
			wantMode: allocationModeDRA,
			wantNode: "node1",
		},
		{
			name:      "explicit dra fails without DeviceClass",
			requested: allocationModeDRA,
			objects:   []runtime.Object{gpuNode("node1", 8, true, false)},
			wantErr:   "allocation mode 'dra' requested but DRA is not usable",
		},
		{
			name:      "explicit dra fails when slices exist but their node is NotReady",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				gpuResourceSlice("down-gpu", "gpu.nvidia.com", "down-node", 8),
				gpuNode("down-node", 0, false, false),
			},
			wantErr: "none is reachable from a Ready, schedulable node",
		},
		{
			name:      "explicit dra fails when the slice's node does not exist",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				gpuResourceSlice("ghost-gpu", "gpu.nvidia.com", "ghost-node", 8),
			},
			wantErr: "none is reachable from a Ready, schedulable node",
		},
		{
			name:      "explicit device-plugin succeeds with Ready GPU node",
			requested: allocationModeDevicePlugin,
			objects:   []runtime.Object{gpuNode("node1", 8, true, false)},
			wantMode:  allocationModeDevicePlugin,
			wantNode:  "node1",
		},
		{
			name:      "explicit device-plugin fails with no capacity",
			requested: allocationModeDevicePlugin,
			objects:   []runtime.Object{gpuNode("cpu-node", 0, true, false)},
			wantErr:   "no device-plugin capacity found",
		},
		{
			name:      "explicit device-plugin ignores NotReady nodes",
			requested: allocationModeDevicePlugin,
			objects:   []runtime.Object{gpuNode("node1", 8, false, false)},
			wantErr:   "no device-plugin capacity found",
		},
		{
			name:      "explicit device-plugin ignores unschedulable nodes",
			requested: allocationModeDevicePlugin,
			objects:   []runtime.Object{gpuNode("node1", 8, true, true)},
			wantErr:   "no device-plugin capacity found",
		},
		{
			name:      "explicit device-plugin rejects a DRA-backed extended resource",
			requested: allocationModeDevicePlugin,
			objects: []runtime.Object{
				extendedResourceDeviceClass("gpu.nvidia.com", "nvidia.com/gpu"),
				gpuNode("node1", 8, true, false),
			},
			wantErr: "may be satisfied by DRA instead of the device plugin",
		},
		{
			name:      "auto fallback rejects a DRA-backed extended resource",
			requested: allocationModeAuto,
			objects: []runtime.Object{
				// DeviceClass exists but has no backing slices, so DRA is
				// unusable; the device-plugin fallback must still be
				// attributable, and the extendedResourceName mapping makes
				// it ambiguous.
				extendedResourceDeviceClass("gpu.nvidia.com", "nvidia.com/gpu"),
				gpuNode("node1", 8, true, false),
			},
			wantErr: "device-plugin fallback would not be deterministic",
		},
		{
			name:      "explicit dra resolves a nodeSelector slice against a matching Ready node",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				selectorResourceSlice("sel-gpu", "gpu.nvidia.com", gpuLabelSelector("accelerator", "gpu"), 4),
				labeledNode("gpu-node", map[string]string{"accelerator": "gpu"}, true),
				labeledNode("cpu-node", nil, true),
			},
			wantMode: allocationModeDRA,
			wantNode: "gpu-node",
		},
		{
			name:      "explicit dra fails when the nodeSelector matches no usable node",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				selectorResourceSlice("sel-gpu", "gpu.nvidia.com", gpuLabelSelector("accelerator", "gpu"), 4),
				labeledNode("gpu-node", map[string]string{"accelerator": "gpu"}, false),
				labeledNode("cpu-node", nil, true),
			},
			wantErr: "none is reachable from a Ready, schedulable node",
		},
		{
			name:      "explicit dra resolves an allNodes slice to a usable node",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				allNodesResourceSlice("all-gpu", "gpu.nvidia.com", 4),
				gpuNode("node1", 0, true, false),
			},
			wantMode: allocationModeDRA,
			wantNode: "node1",
		},
		{
			name:      "explicit dra fails for an allNodes slice with no usable nodes",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				allNodesResourceSlice("all-gpu", "gpu.nvidia.com", 4),
				gpuNode("down-node", 0, false, false),
			},
			wantErr: "none is reachable from a Ready, schedulable node",
		},
		{
			name:      "explicit dra uses only the highest pool generation",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				// The stale generation points at a node that no longer
				// exists; only the current generation's node is eligible.
				pooledResourceSlice("gen1", "gpu.nvidia.com", "gone-node", "pool-a", 1, 8, 1),
				pooledResourceSlice("gen2", "gpu.nvidia.com", "node1", "pool-a", 2, 8, 1),
				gpuNode("node1", 0, true, false),
			},
			wantMode: allocationModeDRA,
			wantNode: "node1",
		},
		{
			name:      "explicit dra ignores devices only advertised by an obsolete pool generation",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				pooledResourceSlice("gen1", "gpu.nvidia.com", "node1", "pool-a", 1, 8, 1),
				pooledResourceSlice("gen2", "gpu.nvidia.com", "node1", "pool-a", 2, 0, 1),
				gpuNode("node1", 0, true, false),
			},
			wantErr: "current-generation ResourceSlice with allocatable devices",
		},
		{
			name:      "explicit dra ignores an incomplete current-generation pool",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				// The pool declares two slices at this generation but only
				// one has been observed; the scheduler ignores incomplete
				// pools, so the probe must too.
				pooledResourceSlice("part1", "gpu.nvidia.com", "node1", "pool-a", 1, 8, 2),
				gpuNode("node1", 0, true, false),
			},
			wantErr: "current-generation ResourceSlice with allocatable devices",
		},
		{
			name:      "explicit dra accepts a complete multi-slice pool",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				pooledResourceSlice("part1", "gpu.nvidia.com", "node1", "pool-a", 1, 8, 2),
				pooledResourceSlice("part2", "gpu.nvidia.com", "node1", "pool-a", 1, 8, 2),
				gpuNode("node1", 0, true, false),
			},
			wantMode: allocationModeDRA,
			wantNode: "node1",
		},
		{
			name:      "explicit dra fails when every device is taint-blocked",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				taintedResourceSlice("tainted", "gpu.nvidia.com", "node1"),
				gpuNode("node1", 0, true, false),
			},
			wantErr: "current-generation ResourceSlice with allocatable devices",
		},
		{
			name:      "allNodes slice prefers a node advertising the extended resource",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				allNodesResourceSlice("all-gpu", "gpu.nvidia.com", 4),
				// "aaa-cpu" sorts first but advertises no accelerators;
				// the pick must prefer the accelerator-bearing node.
				labeledNode("aaa-cpu", nil, true),
				gpuNode("zzz-gpu", 8, true, false),
			},
			wantMode: allocationModeDRA,
			wantNode: "zzz-gpu",
		},
		{
			name:      "explicit dra resolves per-device node selection",
			requested: allocationModeDRA,
			objects: []runtime.Object{
				gpuDeviceClass("gpu.nvidia.com"),
				perDeviceResourceSlice("pd-gpu", "gpu.nvidia.com", "node1"),
				gpuNode("node1", 0, true, false),
			},
			wantMode: allocationModeDRA,
			wantNode: "node1",
		},
		{
			name:      "invalid mode value is rejected",
			requested: "bogus",
			wantErr:   `invalid -allocation-mode "bogus"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientset(tt.objects...)
			mode, node, err := detectAllocationMode(context.Background(), client, tt.requested, cfg, t.Logf)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got mode %q", tt.wantErr, mode)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mode != tt.wantMode {
				t.Fatalf("mode = %q, want %q", mode, tt.wantMode)
			}
			if node != tt.wantNode {
				t.Fatalf("node = %q, want %q", node, tt.wantNode)
			}
		})
	}
}

// TestExtendedResourceGuardFailsClosed verifies that only "DRA API not
// served" lets the device-plugin attribution guard pass; every other
// DeviceClass-list failure (Forbidden, transient API error) must propagate
// instead of being read as proof that no DRA mapping exists.
func TestExtendedResourceGuardFailsClosed(t *testing.T) {
	cfg := nvidiaConfig(t)
	deviceClassGR := schema.GroupResource{Group: "resource.k8s.io", Resource: "deviceclasses"}

	tests := []struct {
		name     string
		listErr  error
		wantMode string
		wantErr  string
	}{
		{
			name:     "API-not-served NotFound passes the guard",
			listErr:  apierrors.NewNotFound(deviceClassGR, ""),
			wantMode: allocationModeDevicePlugin,
		},
		{
			name:    "Forbidden fails closed",
			listErr: apierrors.NewForbidden(deviceClassGR, "", errors.New("rbac denied")),
			wantErr: "failed to list DeviceClasses",
		},
		{
			name:    "transient API error fails closed",
			listErr: errors.New("etcdserver: request timed out"),
			wantErr: "failed to list DeviceClasses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientset(gpuNode("node1", 8, true, false))
			client.PrependReactor("list", "deviceclasses", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, tt.listErr
			})
			mode, _, err := detectAllocationMode(context.Background(), client, allocationModeDevicePlugin, cfg, t.Logf)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got mode=%q err=%v", tt.wantErr, mode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mode != tt.wantMode {
				t.Fatalf("mode = %q, want %q", mode, tt.wantMode)
			}
		})
	}
}

func TestLookupAcceleratorConfig(t *testing.T) {
	if _, err := lookupAcceleratorConfig("nvidia"); err != nil {
		t.Errorf("nvidia should be supported: %v", err)
	}
	if _, err := lookupAcceleratorConfig("nvida"); err == nil {
		t.Error("expected error for unsupported accelerator type")
	} else if !strings.Contains(err.Error(), "unsupported accelerator type") {
		t.Errorf("error %q should mention the unsupported type", err.Error())
	}
}

func TestBuildTestPod(t *testing.T) {
	cfg := nvidiaConfig(t)
	containers := func() []corev1.Container {
		return []corev1.Container{
			acceleratorProbingContainer("authorized", cfg),
			acceleratorProbingContainer("unauthorized", cfg),
		}
	}

	t.Run("dra grant wires the claim to the first container only", func(t *testing.T) {
		pod, err := buildTestPod("ns", "p", containers(), testPodConfig{grantAccelerator: true, mode: allocationModeDRA, cfg: cfg})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pod.Spec.ResourceClaims) != 1 || pod.Spec.ResourceClaims[0].ResourceClaimTemplateName == nil ||
			*pod.Spec.ResourceClaims[0].ResourceClaimTemplateName != testResourceTemplateName {
			t.Errorf("pod-level resourceClaims not wired to template: %+v", pod.Spec.ResourceClaims)
		}
		if len(pod.Spec.Containers[0].Resources.Claims) != 1 {
			t.Errorf("first container should carry the claim, got %+v", pod.Spec.Containers[0].Resources.Claims)
		}
		if len(pod.Spec.Containers[1].Resources.Claims) != 0 || len(pod.Spec.Containers[1].Resources.Limits) != 0 {
			t.Errorf("second container must not be granted the accelerator: %+v", pod.Spec.Containers[1].Resources)
		}
	})

	t.Run("device-plugin grant sets the extended resource on the first container only", func(t *testing.T) {
		pod, err := buildTestPod("ns", "p", containers(), testPodConfig{grantAccelerator: true, mode: allocationModeDevicePlugin, cfg: cfg})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		q, ok := pod.Spec.Containers[0].Resources.Limits[corev1.ResourceName(cfg.ExtendedResource)]
		if !ok || q.Value() != 1 {
			t.Errorf("first container should request 1 %s, got %+v", cfg.ExtendedResource, pod.Spec.Containers[0].Resources.Limits)
		}
		if len(pod.Spec.ResourceClaims) != 0 {
			t.Errorf("device-plugin mode must not set pod-level resourceClaims: %+v", pod.Spec.ResourceClaims)
		}
		if len(pod.Spec.Containers[1].Resources.Limits) != 0 || len(pod.Spec.Containers[1].Resources.Claims) != 0 {
			t.Errorf("second container must not be granted the accelerator: %+v", pod.Spec.Containers[1].Resources)
		}
	})

	t.Run("no grant leaves both containers without accelerator requests", func(t *testing.T) {
		pod, err := buildTestPod("ns", "p", containers(), testPodConfig{grantAccelerator: false, mode: allocationModeDRA, cfg: cfg})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for i, c := range pod.Spec.Containers {
			if len(c.Resources.Claims) != 0 || len(c.Resources.Limits) != 0 {
				t.Errorf("container %d must not carry accelerator requests: %+v", i, c.Resources)
			}
		}
		if len(pod.Spec.ResourceClaims) != 0 {
			t.Errorf("pod must not carry resourceClaims: %+v", pod.Spec.ResourceClaims)
		}
	})

	t.Run("nodeName pinning is applied", func(t *testing.T) {
		pod, err := buildTestPod("ns", "p", containers(), testPodConfig{mode: allocationModeDevicePlugin, nodeName: "gpu-node-1", cfg: cfg})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pod.Spec.NodeName != "gpu-node-1" {
			t.Errorf("nodeName = %q, want gpu-node-1", pod.Spec.NodeName)
		}
	})

	t.Run("unknown allocation mode is rejected", func(t *testing.T) {
		if _, err := buildTestPod("ns", "p", containers(), testPodConfig{grantAccelerator: true, mode: "bogus", cfg: cfg}); err == nil {
			t.Error("expected error for unknown allocation mode")
		}
	})
}
