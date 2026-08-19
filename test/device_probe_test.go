package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcceleratorProbeCommand(t *testing.T) {
	dir := t.TempDir()
	cfg := nvidiaConfig(t)
	for _, name := range []string{"nvidia0", "nvidia1", "nvidiactl", "nvidia-uvm"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatalf("create test device entry %s: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "nvidia-caps"), 0o700); err != nil {
		t.Fatalf("create test control-device directory: %v", err)
	}

	pattern := filepath.Join(dir, filepath.Base(cfg.DevicePattern))
	output, err := exec.Command("/bin/sh", "-c", acceleratorProbeCommand(pattern)).CombinedOutput()
	if err != nil {
		t.Fatalf("run accelerator probe: %v: %s", err, output)
	}
	if got, want := strings.TrimSpace(string(output)), "RESULT: ACCELERATOR_COUNT=2"; got != want {
		t.Fatalf("probe output = %q, want %q", got, want)
	}

	emptyPattern := filepath.Join(dir, "missing[0-9]*")
	output, err = exec.Command("/bin/sh", "-c", acceleratorProbeCommand(emptyPattern)).CombinedOutput()
	if err != nil {
		t.Fatalf("run empty accelerator probe: %v: %s", err, output)
	}
	if got, want := strings.TrimSpace(string(output)), "RESULT: ACCELERATOR_COUNT=0"; got != want {
		t.Fatalf("empty probe output = %q, want %q", got, want)
	}
}

func TestLogsContainExactLine(t *testing.T) {
	tests := []struct {
		name string
		logs string
		want bool
	}{
		{name: "exact count", logs: "setup complete\nRESULT: ACCELERATOR_COUNT=1\n", want: true},
		{name: "surrounding whitespace", logs: "  RESULT: ACCELERATOR_COUNT=1  \n", want: true},
		{name: "over-allocation is not a substring match", logs: "RESULT: ACCELERATOR_COUNT=10\n", want: false},
		{name: "different count", logs: "RESULT: ACCELERATOR_COUNT=0\n", want: false},
		{name: "missing result", logs: "setup complete\n", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logsContainExactLine(tt.logs, "RESULT: ACCELERATOR_COUNT=1"); got != tt.want {
				t.Errorf("logsContainExactLine() = %t, want %t", got, tt.want)
			}
		})
	}
}
