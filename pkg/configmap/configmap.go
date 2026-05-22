package configmap

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Script represents a script loaded from ConfigMap.
type Script struct {
	Name     string
	Content  string
	Schedule string
}

// Loader handles loading scripts from Kubernetes ConfigMap.
type Loader struct {
	clientset  *kubernetes.Clientset
	namespace  string
	configMap  string
}

// NewLoader creates a new ConfigMap loader.
func NewLoader(namespace, configMapName string) (*Loader, error) {
	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return &Loader{
		clientset: clientset,
		namespace: namespace,
		configMap: configMapName,
	}, nil
}

// LoadScripts loads all scripts from the ConfigMap.
func (l *Loader) LoadScripts(ctx context.Context) ([]*Script, error) {
	cm, err := l.clientset.CoreV1().ConfigMaps(l.namespace).Get(ctx, l.configMap, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	var scripts []*Script
	for name, content := range cm.Data {
		schedule, err := extractSchedule(content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to extract schedule from script %s: %v\n", name, err)
			continue
		}

		scripts = append(scripts, &Script{
			Name:     name,
			Content:  content,
			Schedule: schedule,
		})
	}

	return scripts, nil
}

// extractSchedule extracts the cron schedule from the first commented line.
// Expected format: "# * * * * *" or "#!/bin/bash\n# * * * * *"
func extractSchedule(content string) (string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 1 {
		return "", fmt.Errorf("empty script content")
	}

	// Skip shebang line if present
	startIdx := 0
	if strings.HasPrefix(lines[0], "#!") {
		startIdx = 1
	}

	// Find the first comment line that looks like a cron expression
	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "#") {
			// Remove the # and whitespace
			schedule := strings.TrimSpace(line[1:])
			// Basic validation: cron expressions have 5 fields separated by spaces
			fields := strings.Fields(schedule)
			if len(fields) == 5 {
				return schedule, nil
			}
		}
	}

	return "", fmt.Errorf("no valid cron schedule found in script")
}

// WatchScripts watches for changes to the ConfigMap and returns updates.
func (l *Loader) WatchScripts(ctx context.Context) (<-chan []*Script, error) {
	updates := make(chan []*Script)

	go func() {
		defer close(updates)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				scripts, err := l.LoadScripts(ctx)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error loading scripts: %v\n", err)
					continue
				}
				updates <- scripts
			}
		}
	}()

	return updates, nil
}
