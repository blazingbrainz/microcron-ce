package configmap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Script struct {
	Name     string
	Content  string
	Schedule string
}

type Loader struct {
	mountPath string
}

// NewLoader creates a new ConfigMap loader that reads from a mounted volume.
func NewLoader(namespace, configMapName string) (*Loader, error) {
	// ConfigMap is mounted at /etc/microcron-ce/scripts
	mountPath := "/etc/microcron-ce/scripts"

	// Verify the mount path exists
	if _, err := os.Stat(mountPath); err != nil {
		return nil, fmt.Errorf("ConfigMap mount path %s not found: %w", mountPath, err)
	}

	return &Loader{
		mountPath: mountPath,
	}, nil
}

// LoadScripts loads all scripts from the mounted ConfigMap volume.
func (l *Loader) LoadScripts(ctx context.Context) ([]*Script, error) {
	entries, err := os.ReadDir(l.mountPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ConfigMap mount path: %w", err)
	}

	var scripts []*Script
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(l.mountPath, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to read script file %s: %v\n", entry.Name(), err)
			continue
		}

		contentStr := string(content)
		schedule, err := extractSchedule(contentStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to extract schedule from script %s: %v\n", entry.Name(), err)
			continue
		}

		scripts = append(scripts, &Script{
			Name:     entry.Name(),
			Content:  contentStr,
			Schedule: schedule,
		})
	}

	return scripts, nil
}

// extractSchedule extracts the cron schedule from the first commented line.
func extractSchedule(content string) (string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 1 {
		return "", fmt.Errorf("empty script content")
	}

	startIdx := 0
	if strings.HasPrefix(lines[0], "#!") {
		startIdx = 1
	}

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "#") {
			schedule := strings.TrimSpace(line[1:])
			fields := strings.Fields(schedule)
			if len(fields) == 5 {
				return schedule, nil
			}
		}
	}

	return "", fmt.Errorf("no valid cron schedule found in script")
}

// WatchScripts watches for changes to the ConfigMap by polling the mounted directory.
func (l *Loader) WatchScripts(ctx context.Context) (<-chan []*Script, error) {
	updates := make(chan []*Script)

	go func() {
		defer close(updates)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		var lastScripts []*Script

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

				// Only send update if scripts changed
				if !scriptsEqual(lastScripts, scripts) {
					updates <- scripts
					lastScripts = scripts
				}
			}
		}
	}()

	return updates, nil
}

// scriptsEqual checks if two script slices are equal.
func scriptsEqual(a, b []*Script) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name || a[i].Content != b[i].Content || a[i].Schedule != b[i].Schedule {
			return false
		}
	}
	return true
}
