package configmap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SecretRef struct {
	Name string
	Keys []string
}

type Script struct {
	Name       string
	Content    string
	Schedule   string
	SecretRefs []SecretRef
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
		schedule, scheduleLineIdx, err := extractScheduleWithIdx(contentStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to extract schedule from script %s: %v\n", entry.Name(), err)
			continue
		}

		script := &Script{
			Name:     entry.Name(),
			Content:  contentStr,
			Schedule: schedule,
		}

		if scheduleLineIdx >= 0 {
			secretRef := extractSecretRef(contentStr, scheduleLineIdx)
			if secretRef != nil {
				script.SecretRefs = []SecretRef{*secretRef}
			}
		}

		scripts = append(scripts, script)
	}

	return scripts, nil
}

// extractSchedule extracts the cron schedule from the first commented line and returns its line index.
func extractScheduleWithIdx(content string) (string, int, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 1 {
		return "", -1, fmt.Errorf("empty script content")
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
				return schedule, i, nil
			}
		}
	}

	return "", -1, fmt.Errorf("no valid cron schedule found in script")
}

// extractSchedule extracts the cron schedule from the first commented line.
func extractSchedule(content string) (string, error) {
	schedule, _, err := extractScheduleWithIdx(content)
	return schedule, err
}

// extractSecretRef extracts the optional secret reference from the line after the schedule.
// Format: # secretname: key1, key2, key3
func extractSecretRef(content string, scheduleLineIdx int) *SecretRef {
	lines := strings.Split(content, "\n")

	if scheduleLineIdx < 0 || scheduleLineIdx >= len(lines)-1 {
		return nil
	}

	for i := scheduleLineIdx + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			return nil
		}

		content := strings.TrimSpace(line[1:])
		if !strings.Contains(content, ":") {
			return nil
		}

		parts := strings.SplitN(content, ":", 2)
		if len(parts) != 2 {
			return nil
		}

		name := strings.TrimSpace(parts[0])
		keysStr := strings.TrimSpace(parts[1])

		if name == "" || keysStr == "" {
			return nil
		}

		var keys []string
		for _, key := range strings.Split(keysStr, ",") {
			key = strings.TrimSpace(key)
			if key != "" {
				keys = append(keys, key)
			}
		}

		if len(keys) == 0 {
			return nil
		}

		return &SecretRef{
			Name: name,
			Keys: keys,
		}
	}

	return nil
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
		if !secretRefsEqual(a[i].SecretRefs, b[i].SecretRefs) {
			return false
		}
	}
	return true
}

// secretRefsEqual checks if two SecretRef slices are equal.
func secretRefsEqual(a, b []SecretRef) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name {
			return false
		}
		if len(a[i].Keys) != len(b[i].Keys) {
			return false
		}
		for j := range a[i].Keys {
			if a[i].Keys[j] != b[i].Keys[j] {
				return false
			}
		}
	}
	return true
}
