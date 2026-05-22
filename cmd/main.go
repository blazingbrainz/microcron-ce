package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blazingbrainz/microcron-ce/pkg/configmap"
	"github.com/blazingbrainz/microcron-ce/pkg/cron"
	"github.com/blazingbrainz/microcron-ce/pkg/logger"
)

func main() {
	// Parse command-line flags
	namespace := flag.String("namespace", "default", "Kubernetes namespace to watch ConfigMaps from")
	configMapName := flag.String("configmap", "microcron-scripts", "Name of the ConfigMap containing scripts")
	logDir := flag.String("log-dir", "/var/log/microcron-ce", "Directory for log files")
	retentionDays := flag.Int("retention-days", 7, "Number of days to retain log files")
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	// Initialize logger
	log, err := logger.NewRotatingLogger(*logDir, *retentionDays)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	log.Log("========== Starting microcron-ce ==========")
	log.Log(fmt.Sprintf("Namespace: %s", *namespace))
	log.Log(fmt.Sprintf("ConfigMap: %s", *configMapName))
	log.Log(fmt.Sprintf("Log Directory: %s", *logDir))
	log.Log(fmt.Sprintf("Retention Days: %d", *retentionDays))
	if *debug {
		log.Log("Debug mode: ENABLED")
	}

	// Initialize ConfigMap loader
	loader, err := configmap.NewLoader(*namespace, *configMapName)
	if err != nil {
		log.Log(fmt.Sprintf("Failed to initialize ConfigMap loader: %v", err))
		os.Exit(1)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize cron scheduler
	scheduler := cron.NewScheduler(log)
	scheduler.Start()

	// Load initial scripts
	scripts, err := loader.LoadScripts(ctx)
	if err != nil {
		log.Log(fmt.Sprintf("Failed to load scripts from ConfigMap: %v", err))
		os.Exit(1)
	}

	// Add scripts to scheduler
	for _, script := range scripts {
		if err := scheduler.AddJob(script); err != nil {
			log.Log(fmt.Sprintf("Failed to add job %s: %v", script.Name, err))
		}
	}

	log.Log(fmt.Sprintf("Loaded %d scripts from ConfigMap", len(scripts)))

	// Start watching for ConfigMap updates
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Log("========== Cron scheduler ready ==========")

	// Main loop - watch for ConfigMap updates and handle signals
	for {
		select {
		case <-ticker.C:
			// Reload scripts from ConfigMap periodically
			updatedScripts, err := loader.LoadScripts(ctx)
			if err != nil {
				log.Log(fmt.Sprintf("Error reloading scripts: %v", err))
				continue
			}

			// Update scheduler with new scripts
			if err := scheduler.UpdateJobs(updatedScripts); err != nil {
				log.Log(fmt.Sprintf("Error updating scheduler: %v", err))
			}

		case <-sigChan:
			log.Log("Received termination signal, shutting down...")
			scheduler.Stop()
			log.Log("========== Shutting down microcron-ce ==========")
			os.Exit(0)
		}
	}
}
