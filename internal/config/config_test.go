package config

import (
	"os"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary test config file
	content := []byte(`{
		"queues": [
			{
				"name": "test_queue_1",
				"size": 100,
				"max_sub": 10
			},
			{
				"name": "test_queue_2",
				"size": 50,
				"max_sub": 5
			}
		],
		"addr": "localhost:9090"
	}`)
	tmpfile, err := os.CreateTemp("", "test_config.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }() // clean up

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	// Test successful loading
	t.Run("success", func(t *testing.T) {
		cfg, err := Load(tmpfile.Name())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expectedCfg := &Config{
			Queues: []QueueConfig{
				{Name: "test_queue_1", Size: 100, MaxSub: 10},
				{Name: "test_queue_2", Size: 50, MaxSub: 5},
			},
			Addr: "localhost:9090",
		}

		if !reflect.DeepEqual(cfg, expectedCfg) {
			t.Errorf("expected config %+v, got %+v", expectedCfg, cfg)
		}
	})

	// Test error case: file not found
	t.Run("file not found", func(t *testing.T) {
		_, err := Load("non_existent_file.json")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
	})

	// Test error case: invalid json
	t.Run("invalid json", func(t *testing.T) {
		invalidContent := []byte(`{ "addr": "localhost:9090", "queues": [ { "name": "q1" }`)
		invalidTmpFile, err := os.CreateTemp("", "invalid_config.json")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer func() { _ = os.Remove(invalidTmpFile.Name()) }()

		if _, err := invalidTmpFile.Write(invalidContent); err != nil {
			t.Fatalf("failed to write to temp file: %v", err)
		}
		if err := invalidTmpFile.Close(); err != nil {
			t.Fatalf("failed to close temp file: %v", err)
		}

		_, err = Load(invalidTmpFile.Name())
		if err == nil {
			t.Fatal("expected an error for invalid json, got nil")
		}
	})
}
