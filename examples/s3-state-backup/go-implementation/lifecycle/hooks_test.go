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

package lifecycle

import (
	"context"
	"errors"
	"testing"
)

func TestManager_Register(t *testing.T) {
	manager := NewManager()

	hook := NewFuncHook("test-hook", func(ctx context.Context, event *Event) error {
		return nil
	})

	manager.Register(HookPreCreate, hook)

	if !manager.HasHooks(HookPreCreate) {
		t.Error("Expected hook to be registered for HookPreCreate")
	}

	if manager.HasHooks(HookPostCreate) {
		t.Error("Expected no hooks for HookPostCreate")
	}
}

func TestManager_Execute_Success(t *testing.T) {
	manager := NewManager()

	executed := false
	hook := NewFuncHook("test-hook", func(ctx context.Context, event *Event) error {
		executed = true
		if event.InstanceName != "test-instance" {
			t.Errorf("Expected instance name 'test-instance', got '%s'", event.InstanceName)
		}
		return nil
	})

	manager.Register(HookPreCreate, hook)

	event := &Event{
		Type:         HookPreCreate,
		InstanceName: "test-instance",
		Username:     "testuser",
	}

	err := manager.Execute(context.Background(), event)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !executed {
		t.Error("Hook was not executed")
	}
}

func TestManager_Execute_PreHookFailure(t *testing.T) {
	manager := NewManager()

	expectedErr := errors.New("hook failed")
	hook := NewFuncHook("failing-hook", func(ctx context.Context, event *Event) error {
		return expectedErr
	})

	manager.Register(HookPreCreate, hook)

	event := &Event{
		Type:         HookPreCreate,
		InstanceName: "test-instance",
	}

	err := manager.Execute(context.Background(), event)
	if err == nil {
		t.Error("Expected error from pre-hook failure")
	}
}

func TestManager_Execute_PostHookFailure(t *testing.T) {
	manager := NewManager()

	expectedErr := errors.New("hook failed")
	hook := NewFuncHook("failing-hook", func(ctx context.Context, event *Event) error {
		return expectedErr
	})

	manager.Register(HookPostCreate, hook)

	event := &Event{
		Type:         HookPostCreate,
		InstanceName: "test-instance",
	}

	// Post-hook failures should not return error (logged but continue)
	err := manager.Execute(context.Background(), event)
	if err != nil {
		t.Errorf("Post-hook failure should not return error, got: %v", err)
	}
}

func TestManager_Execute_MultipleHooks(t *testing.T) {
	manager := NewManager()

	execOrder := []string{}

	hook1 := NewFuncHook("hook-1", func(ctx context.Context, event *Event) error {
		execOrder = append(execOrder, "hook-1")
		return nil
	})

	hook2 := NewFuncHook("hook-2", func(ctx context.Context, event *Event) error {
		execOrder = append(execOrder, "hook-2")
		return nil
	})

	manager.Register(HookPreCreate, hook1)
	manager.Register(HookPreCreate, hook2)

	event := &Event{
		Type:         HookPreCreate,
		InstanceName: "test-instance",
	}

	err := manager.Execute(context.Background(), event)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(execOrder) != 2 {
		t.Errorf("Expected 2 hooks executed, got %d", len(execOrder))
	}

	if execOrder[0] != "hook-1" || execOrder[1] != "hook-2" {
		t.Errorf("Hooks executed in wrong order: %v", execOrder)
	}
}

func TestManager_ExecuteAsync(t *testing.T) {
	manager := NewManager()

	executedChan := make(chan bool, 1)

	hook := NewFuncHook("async-hook", func(ctx context.Context, event *Event) error {
		executedChan <- true
		return nil
	})

	manager.Register(HookPostCreate, hook)

	event := &Event{
		Type:         HookPostCreate,
		InstanceName: "test-instance",
	}

	// ExecuteAsync doesn't block
	manager.ExecuteAsync(event)

	// Wait for async execution
	<-executedChan
}

func TestIsPreHook(t *testing.T) {
	tests := []struct {
		hookType HookType
		isPre    bool
	}{
		{HookPreCreate, true},
		{HookPreDelete, true},
		{HookPreStart, true},
		{HookPreStop, true},
		{HookPostCreate, false},
		{HookPostDelete, false},
		{HookPostStart, false},
		{HookPostStop, false},
	}

	for _, tt := range tests {
		result := isPreHook(tt.hookType)
		if result != tt.isPre {
			t.Errorf("isPreHook(%s) = %v, want %v", tt.hookType, result, tt.isPre)
		}
	}
}
