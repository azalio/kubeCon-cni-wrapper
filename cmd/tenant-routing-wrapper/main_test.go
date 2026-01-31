package main

import (
	"testing"
)

func TestParseCNIArgs_ValidArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          string
		wantPodName   string
		wantNamespace string
	}{
		{
			name:          "standard kubelet args",
			args:          "K8S_POD_NAME=nginx;K8S_POD_NAMESPACE=default;K8S_POD_UID=abc-123",
			wantPodName:   "nginx",
			wantNamespace: "default",
		},
		{
			name:          "minimal args",
			args:          "K8S_POD_NAME=web;K8S_POD_NAMESPACE=production",
			wantPodName:   "web",
			wantNamespace: "production",
		},
		{
			name:          "args with extra fields",
			args:          "IgnoreUnknown=true;K8S_POD_NAMESPACE=kube-system;K8S_POD_NAME=coredns;K8S_POD_UID=xyz-789",
			wantPodName:   "coredns",
			wantNamespace: "kube-system",
		},
		{
			name:          "args with special characters in values",
			args:          "K8S_POD_NAME=my-app-v1.0;K8S_POD_NAMESPACE=my-namespace",
			wantPodName:   "my-app-v1.0",
			wantNamespace: "my-namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podName, podNamespace, err := parseCNIArgs(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if podName != tt.wantPodName {
				t.Errorf("podName = %q, want %q", podName, tt.wantPodName)
			}
			if podNamespace != tt.wantNamespace {
				t.Errorf("podNamespace = %q, want %q", podNamespace, tt.wantNamespace)
			}
		})
	}
}

func TestParseCNIArgs_Empty(t *testing.T) {
	_, _, err := parseCNIArgs("")
	if err == nil {
		t.Fatal("expected error for empty CNI_ARGS")
	}
	if err.Error() != "CNI_ARGS is empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseCNIArgs_MissingPodName(t *testing.T) {
	_, _, err := parseCNIArgs("K8S_POD_NAMESPACE=default")
	if err == nil {
		t.Fatal("expected error for missing pod name")
	}
	if err.Error() != "K8S_POD_NAME not found in CNI_ARGS" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseCNIArgs_MissingNamespace(t *testing.T) {
	_, _, err := parseCNIArgs("K8S_POD_NAME=nginx")
	if err == nil {
		t.Fatal("expected error for missing namespace")
	}
	if err.Error() != "K8S_POD_NAMESPACE not found in CNI_ARGS" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseCNIArgs_MalformedPairs(t *testing.T) {
	// Malformed pairs should be skipped, but required fields still need to be present
	_, _, err := parseCNIArgs("malformed;K8S_POD_NAME=nginx;K8S_POD_NAMESPACE=default;also_malformed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// But if all pairs are malformed and required fields missing, should fail
	_, _, err = parseCNIArgs("malformed;broken;invalid")
	if err == nil {
		t.Fatal("expected error for all malformed pairs")
	}
}

func TestParseCNIArgs_EmptyValues(t *testing.T) {
	// Empty pod name value
	_, _, err := parseCNIArgs("K8S_POD_NAME=;K8S_POD_NAMESPACE=default")
	if err == nil {
		t.Fatal("expected error for empty pod name value")
	}

	// Empty namespace value
	_, _, err = parseCNIArgs("K8S_POD_NAME=nginx;K8S_POD_NAMESPACE=")
	if err == nil {
		t.Fatal("expected error for empty namespace value")
	}
}

func TestParseCNIArgs_EqualsInValue(t *testing.T) {
	// Values can contain equals signs (rare but valid)
	podName, podNamespace, err := parseCNIArgs("K8S_POD_NAME=my=pod;K8S_POD_NAMESPACE=my=ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if podName != "my=pod" {
		t.Errorf("podName = %q, want %q", podName, "my=pod")
	}
	if podNamespace != "my=ns" {
		t.Errorf("podNamespace = %q, want %q", podNamespace, "my=ns")
	}
}
