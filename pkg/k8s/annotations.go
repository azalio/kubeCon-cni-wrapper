package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
)

// K8sAPITimeout is the maximum time allowed for Kubernetes API calls
// CNI operations are time-sensitive; prevents hanging if API is slow/unreachable
const K8sAPITimeout = 5 * time.Second

// ValidFwmarkValues defines the allowed fwmark values for tenant routing
var ValidFwmarkValues = map[string]bool{
	"0x10": true, // Tenant A
	"0x20": true, // Tenant B
}

// GetFwmark retrieves the fwmark annotation value with pod → namespace fallback.
//
// Resolution order:
//  1. Check pod.Annotations[annotationKey]
//  2. If not found, check namespace.Annotations[annotationKey]
//  3. If still not found, return empty string (valid no-op case)
//
// Returns:
//   - fwmark value ('0x10', '0x20', or '') on success
//   - error if pod/namespace API calls fail or fwmark value is invalid
func GetFwmark(clientset kubernetes.Interface, podName, podNamespace, annotationKey string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), K8sAPITimeout)
	defer cancel()

	// Fetch pod
	pod, err := clientset.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("pod %s/%s not found: %w", podNamespace, podName, err)
		}
		return "", fmt.Errorf("failed to get pod %s/%s: %w", podNamespace, podName, err)
	}

	// Check pod annotation first
	if fwmark, ok := pod.Annotations[annotationKey]; ok {
		if err := validateFwmark(fwmark); err != nil {
			return "", fmt.Errorf("invalid fwmark in pod annotation: %w", err)
		}
		return fwmark, nil
	}

	// Fallback to namespace annotation
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, podNamespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("namespace %s not found: %w", podNamespace, err)
		}
		return "", fmt.Errorf("failed to get namespace %s: %w", podNamespace, err)
	}

	if fwmark, ok := ns.Annotations[annotationKey]; ok {
		if err := validateFwmark(fwmark); err != nil {
			return "", fmt.Errorf("invalid fwmark in namespace annotation: %w", err)
		}
		return fwmark, nil
	}

	// Both annotations missing - valid no-op case
	return "", nil
}

// validateFwmark checks if the fwmark value is in the allowed set
func validateFwmark(fwmark string) error {
	if !ValidFwmarkValues[fwmark] {
		return fmt.Errorf("fwmark value '%s' not in allowed set (0x10, 0x20)", fwmark)
	}
	return nil
}
