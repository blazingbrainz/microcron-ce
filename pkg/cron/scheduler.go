package cron

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blazingbrainz/microcron-ce/pkg/configmap"
	"github.com/blazingbrainz/microcron-ce/pkg/executor"
	"github.com/blazingbrainz/microcron-ce/pkg/logger"
	cronlib "github.com/robfig/cron/v3"
)

const secretMountBase = "/etc/microcron-ce/secrets"

// loadSecretEnvVars loads secret values from mounted volumes and returns them as a map.
func loadSecretEnvVars(logger *logger.RotatingLogger, refs []configmap.SecretRef) map[string]string {
	envVars := make(map[string]string)

	for _, ref := range refs {
		mountPath := filepath.Join(secretMountBase, ref.Name)
		for _, key := range ref.Keys {
			filePath := filepath.Join(mountPath, key)
			data, err := os.ReadFile(filePath)
			if err != nil {
				logger.Log(fmt.Sprintf("Warning: Failed to load secret %s key %s: %v", ref.Name, key, err))
				continue
			}
			envVars[key] = strings.TrimSpace(string(data))
		}
	}

	return envVars
}

// Scheduler manages cron job scheduling and execution.
type Scheduler struct {
	cron   *cronlib.Cron
	logger *logger.RotatingLogger
	jobs   map[string]cronlib.EntryID // Map script name to cron entry ID
	mu     sync.Mutex
}

// NewScheduler creates a new cron scheduler.
func NewScheduler(log *logger.RotatingLogger) *Scheduler {
	return &Scheduler{
		cron:   cronlib.New(),
		logger: log,
		jobs:   make(map[string]cronlib.EntryID),
	}
}

// Start starts the cron scheduler.
func (s *Scheduler) Start() {
	s.cron.Start()
	s.logger.Log("Cron scheduler started")
}

// Stop stops the cron scheduler.
func (s *Scheduler) Stop() context.Context {
	ctx := s.cron.Stop()
	s.logger.Log("Cron scheduler stopped")
	return ctx
}

// AddJob adds a script as a cron job.
func (s *Scheduler) AddJob(script *configmap.Script) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing job if present
	if entryID, exists := s.jobs[script.Name]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, script.Name)
	}

	// Create a job function that captures the script content and secrets
	jobFunc := func() {
		secretEnv := loadSecretEnvVars(s.logger, script.SecretRefs)
		result := executor.Execute(script.Name, script.Content, secretEnv)
		s.logger.Log(executor.FormatResult(result))
	}

	// Add the job to the cron schedule
	entryID, err := s.cron.AddFunc(script.Schedule, jobFunc)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to add cron job %s with schedule %s: %v", script.Name, script.Schedule, err)
		s.logger.Log(errorMsg)
		return err
	}

	s.jobs[script.Name] = entryID
	successMsg := fmt.Sprintf("Added cron job: %s (Schedule: %s)", script.Name, script.Schedule)
	s.logger.Log(successMsg)
	return nil
}

// RemoveJob removes a cron job by script name.
func (s *Scheduler) RemoveJob(scriptName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobs[scriptName]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, scriptName)
		s.logger.Log(fmt.Sprintf("Removed cron job: %s", scriptName))
	}
}

// UpdateJobs updates the scheduler with a new set of scripts.
func (s *Scheduler) UpdateJobs(scripts []*configmap.Script) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Build a map of current scripts
	scriptMap := make(map[string]*configmap.Script)
	for _, script := range scripts {
		scriptMap[script.Name] = script
	}

	// Remove jobs that are no longer in the scripts
	for scriptName, entryID := range s.jobs {
		if _, exists := scriptMap[scriptName]; !exists {
			s.cron.Remove(entryID)
			delete(s.jobs, scriptName)
			s.logger.Log(fmt.Sprintf("Removed cron job: %s", scriptName))
		}
	}

	// Add new or updated jobs
	for _, script := range scripts {
		if entryID, exists := s.jobs[script.Name]; exists {
			// Job already exists, check if schedule changed
			currentEntry := s.cron.Entry(entryID)
			if currentEntry.Valid() {
				// For simplicity, always update (remove and re-add)
				s.cron.Remove(entryID)
				delete(s.jobs, script.Name)
			}
		}

		// Add the job
		jobFunc := func(scriptContent string, scriptName string, secretRefs []configmap.SecretRef) func() {
			return func() {
				secretEnv := loadSecretEnvVars(s.logger, secretRefs)
				result := executor.Execute(scriptName, scriptContent, secretEnv)
				s.logger.Log(executor.FormatResult(result))
			}
		}(script.Content, script.Name, script.SecretRefs)

		entryID, err := s.cron.AddFunc(script.Schedule, jobFunc)
		if err != nil {
			return err
		}
		s.jobs[script.Name] = entryID
		s.logger.Log(fmt.Sprintf("Updated cron job: %s (Schedule: %s)", script.Name, script.Schedule))
	}

	return nil
}

// GetJobs returns a list of all scheduled jobs.
func (s *Scheduler) GetJobs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var jobs []string
	for scriptName := range s.jobs {
		jobs = append(jobs, scriptName)
	}
	return jobs
}
