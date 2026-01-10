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

// Package lifecycle provides a hooks system for CVD lifecycle events.
// This code should be added to the android-cuttlefish repository.
package lifecycle

import (
	"context"
	"fmt"
	"log"
)

// HookType represents different lifecycle events
type HookType string

const (
	HookPreCreate   HookType = "pre-create"
	HookPostCreate  HookType = "post-create"
	HookPreDelete   HookType = "pre-delete"
	HookPostDelete  HookType = "post-delete"
	HookPreStart    HookType = "pre-start"
	HookPostStart   HookType = "post-start"
	HookPreStop     HookType = "pre-stop"
	HookPostStop    HookType = "post-stop"
)

// Hook represents a lifecycle hook handler
type Hook interface {
	// Execute runs the hook with the given context and event data
	Execute(ctx context.Context, event *Event) error

	// Name returns a descriptive name for the hook (for logging)
	Name() string
}

// Event contains information about the lifecycle event
type Event struct {
	Type         HookType
	InstanceName string
	InstanceDir  string
	Username     string
	Metadata     map[string]string
}

// Manager manages and executes lifecycle hooks
type Manager struct {
	hooks map[HookType][]Hook
}

// NewManager creates a new lifecycle hook manager
func NewManager() *Manager {
	return &Manager{
		hooks: make(map[HookType][]Hook),
	}
}

// Register registers a hook for a specific lifecycle event
func (m *Manager) Register(hookType HookType, hook Hook) {
	if m.hooks[hookType] == nil {
		m.hooks[hookType] = []Hook{}
	}
	m.hooks[hookType] = append(m.hooks[hookType], hook)
	log.Printf("Registered hook '%s' for event '%s'", hook.Name(), hookType)
}

// Execute runs all hooks registered for the given event type
func (m *Manager) Execute(ctx context.Context, event *Event) error {
	hooks := m.hooks[event.Type]
	if len(hooks) == 0 {
		log.Printf("No hooks registered for event '%s'", event.Type)
		return nil
	}

	log.Printf("Executing %d hook(s) for event '%s' (instance: %s)",
		len(hooks), event.Type, event.InstanceName)

	for _, hook := range hooks {
		log.Printf("Running hook '%s' for event '%s'", hook.Name(), event.Type)

		if err := hook.Execute(ctx, event); err != nil {
			log.Printf("Hook '%s' failed: %v", hook.Name(), err)

			// For pre-* hooks, failures should block the operation
			// For post-* hooks, we log but continue
			if isPreHook(event.Type) {
				return fmt.Errorf("hook '%s' failed: %w", hook.Name(), err)
			}

			log.Printf("Continuing despite hook failure (post-hook)")
		} else {
			log.Printf("Hook '%s' completed successfully", hook.Name())
		}
	}

	return nil
}

// ExecuteAsync runs hooks asynchronously (for post-* hooks that don't need to block)
func (m *Manager) ExecuteAsync(event *Event) {
	go func() {
		ctx := context.Background()
		if err := m.Execute(ctx, event); err != nil {
			log.Printf("Async hook execution failed for event '%s': %v", event.Type, err)
		}
	}()
}

// isPreHook returns true if this is a pre-* hook that should block on failure
func isPreHook(hookType HookType) bool {
	return hookType == HookPreCreate ||
	       hookType == HookPreDelete ||
	       hookType == HookPreStart ||
	       hookType == HookPreStop
}

// HasHooks returns true if any hooks are registered for the given type
func (m *Manager) HasHooks(hookType HookType) bool {
	return len(m.hooks[hookType]) > 0
}

// FuncHook is a simple Hook implementation using a function
type FuncHook struct {
	name string
	fn   func(ctx context.Context, event *Event) error
}

// NewFuncHook creates a hook from a function
func NewFuncHook(name string, fn func(ctx context.Context, event *Event) error) Hook {
	return &FuncHook{name: name, fn: fn}
}

func (h *FuncHook) Execute(ctx context.Context, event *Event) error {
	return h.fn(ctx, event)
}

func (h *FuncHook) Name() string {
	return h.name
}
