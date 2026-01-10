// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package integration shows how to integrate lifecycle hooks and S3 state backup
// into the Host Orchestrator (android-cuttlefish repository).
package integration

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	// These would be actual imports in android-cuttlefish
	"github.com/google/cloud-android-orchestration/examples/s3-state-backup/go-implementation/lifecycle"
	"github.com/google/cloud-android-orchestration/examples/s3-state-backup/go-implementation/s3state"
)

// Server is the Host Orchestrator server with lifecycle hooks support
type Server struct {
	router          *mux.Router
	lifecycleHooks  *lifecycle.Manager
	s3StateManager  *s3state.Manager
	instanceBaseDir string
}

// NewServer creates a new Host Orchestrator server
func NewServer() (*Server, error) {
	s := &Server{
		router:          mux.NewRouter(),
		lifecycleHooks:  lifecycle.NewManager(),
		instanceBaseDir: "/home/vsoc-01/cuttlefish_runtime/instances",
	}

	// Initialize S3 state manager if configured
	if err := s.initializeS3StateManager(); err != nil {
		log.Printf("S3 state backup not available: %v", err)
	}

	// Register lifecycle hooks
	s.registerHooks()

	// Setup HTTP routes
	s.setupRoutes()

	return s, nil
}

// initializeS3StateManager sets up S3 state backup from environment variables
func (s *Server) initializeS3StateManager() error {
	bucket := os.Getenv("CVD_S3_BUCKET")
	if bucket == "" {
		return nil // S3 not configured, skip
	}

	config := s3state.Config{
		Enabled: true,
		Bucket:  bucket,
		Prefix:  getEnvOrDefault("CVD_S3_PREFIX", "states"),
		Region:  getEnvOrDefault("AWS_REGION", "us-west-2"),
	}

	manager, err := s3state.NewManager(config)
	if err != nil {
		return err
	}

	s.s3StateManager = manager
	log.Printf("S3 state backup enabled (bucket: %s, region: %s)", config.Bucket, config.Region)
	return nil
}

// registerHooks registers lifecycle hooks for S3 state backup
func (s *Server) registerHooks() {
	// Register S3 state backup hooks if S3 is configured
	if s.s3StateManager != nil {
		// Pre-create: Restore state from S3
		s.lifecycleHooks.Register(lifecycle.HookPreCreate,
			lifecycle.NewFuncHook("s3-restore-state", func(ctx context.Context, event *lifecycle.Event) error {
				return s.s3StateManager.RestoreState(ctx,
					event.Username,
					event.InstanceName,
					event.InstanceDir,
				)
			}))

		// Pre-delete: Backup state to S3
		s.lifecycleHooks.Register(lifecycle.HookPreDelete,
			lifecycle.NewFuncHook("s3-backup-state", func(ctx context.Context, event *lifecycle.Event) error {
				return s.s3StateManager.BackupState(ctx,
					event.Username,
					event.InstanceName,
					event.InstanceDir,
				)
			}))
	}

	// You can register additional hooks here
	// Example: Logging hook
	s.lifecycleHooks.Register(lifecycle.HookPostCreate,
		lifecycle.NewFuncHook("log-creation", func(ctx context.Context, event *lifecycle.Event) error {
			log.Printf("CVD instance created: %s (user: %s)", event.InstanceName, event.Username)
			return nil
		}))

	s.lifecycleHooks.Register(lifecycle.HookPostDelete,
		lifecycle.NewFuncHook("log-deletion", func(ctx context.Context, event *lifecycle.Event) error {
			log.Printf("CVD instance deleted: %s (user: %s)", event.InstanceName, event.Username)
			return nil
		}))
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() {
	s.router.HandleFunc("/cvds", s.CreateCVD).Methods("POST")
	s.router.HandleFunc("/cvds", s.ListCVDs).Methods("GET")
	s.router.HandleFunc("/cvds/{id}", s.DeleteCVD).Methods("DELETE")
	s.router.HandleFunc("/cvds/{id}", s.GetCVD).Methods("GET")
}

// CreateCVDRequest represents a request to create a CVD
type CreateCVDRequest struct {
	EnvConfig map[string]interface{} `json:"env_config"`
}

// CreateCVD handles CVD creation with lifecycle hooks
func (s *Server) CreateCVD(w http.ResponseWriter, r *http.Request) {
	var req CreateCVDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract instance details
	instanceName := extractInstanceName(req.EnvConfig) // You'd implement this
	username := extractUsername(r)                      // Extract from auth header/context
	instanceDir := s.getInstanceDir(instanceName)

	ctx := r.Context()

	// Execute pre-create hooks
	event := &lifecycle.Event{
		Type:         lifecycle.HookPreCreate,
		InstanceName: instanceName,
		InstanceDir:  instanceDir,
		Username:     username,
		Metadata:     make(map[string]string),
	}

	if err := s.lifecycleHooks.Execute(ctx, event); err != nil {
		log.Printf("Pre-create hook failed: %v", err)
		http.Error(w, "Pre-create hook failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create the CVD instance (your existing logic)
	if err := s.createCVDInstance(ctx, instanceName, instanceDir, req.EnvConfig); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute post-create hooks asynchronously
	event.Type = lifecycle.HookPostCreate
	s.lifecycleHooks.ExecuteAsync(event)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":        "success",
		"instance_name": instanceName,
	})
}

// DeleteCVD handles CVD deletion with lifecycle hooks
func (s *Server) DeleteCVD(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["id"]
	username := extractUsername(r)
	instanceDir := s.getInstanceDir(instanceName)

	ctx := r.Context()

	// Execute pre-delete hooks (CRITICAL: backup before deletion!)
	event := &lifecycle.Event{
		Type:         lifecycle.HookPreDelete,
		InstanceName: instanceName,
		InstanceDir:  instanceDir,
		Username:     username,
		Metadata:     make(map[string]string),
	}

	if err := s.lifecycleHooks.Execute(ctx, event); err != nil {
		log.Printf("Pre-delete hook failed: %v", err)
		// Decide whether to block deletion on backup failure
		// For production, you might want to block to prevent data loss
		http.Error(w, "Pre-delete hook failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete the CVD instance (your existing logic)
	if err := s.deleteCVDInstance(ctx, instanceName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute post-delete hooks asynchronously
	event.Type = lifecycle.HookPostDelete
	s.lifecycleHooks.ExecuteAsync(event)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// ListCVDs lists all CVD instances
func (s *Server) ListCVDs(w http.ResponseWriter, r *http.Request) {
	// Your existing implementation
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"cvds": []string{},
	})
}

// GetCVD gets a specific CVD instance
func (s *Server) GetCVD(w http.ResponseWriter, r *http.Request) {
	// Your existing implementation
	w.WriteHeader(http.StatusOK)
}

// Helper methods

func (s *Server) getInstanceDir(instanceName string) string {
	return s.instanceBaseDir + "/" + instanceName
}

func (s *Server) createCVDInstance(ctx context.Context, instanceName, instanceDir string, config map[string]interface{}) error {
	// Your existing CVD creation logic here
	// This would call launch_cvd, etc.
	log.Printf("Creating CVD instance: %s at %s", instanceName, instanceDir)
	return nil
}

func (s *Server) deleteCVDInstance(ctx context.Context, instanceName string) error {
	// Your existing CVD deletion logic here
	log.Printf("Deleting CVD instance: %s", instanceName)
	return nil
}

func extractInstanceName(envConfig map[string]interface{}) string {
	// Extract instance name from env_config
	// This depends on your CVD configuration format
	if name, ok := envConfig["instance_name"].(string); ok {
		return name
	}
	return "cvd-1" // Default
}

func extractUsername(r *http.Request) string {
	// Extract username from request context or headers
	// This depends on your authentication setup
	if username := r.Header.Get("X-CVD-Username"); username != "" {
		return username
	}
	username := os.Getenv("CVD_USERNAME")
	if username == "" {
		username = "default"
	}
	return username
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	log.Printf("Starting Host Orchestrator on %s", addr)
	return http.ListenAndServe(addr, s.router)
}

// Example main function showing how to use this
func ExampleMain() {
	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Run(":2080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
