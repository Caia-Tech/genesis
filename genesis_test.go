package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestDatasetLoader tests the dataset loading functionality
func TestDatasetLoader(t *testing.T) {
	// Create test file
	testFile := "test_dataset.txt"
	testContent := "hello world test data artificial intelligence machine learning"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	config := TrainingConfig{
		DatasetPaths: []string{testFile},
		MaxVocabSize: 1000,
		EmbeddingDim: 64,
		MinWordFreq:  1,
		MaxDocuments: 10,
	}

	t.Run("Basic Loading", func(t *testing.T) {
		loader, err := NewDatasetLoader(config)
		if err != nil {
			t.Fatalf("Failed to create dataset loader: %v", err)
		}

		vocab := loader.GetVocabulary()
		if len(vocab) == 0 {
			t.Error("Vocabulary is empty")
		}

		docs := loader.GetDocuments()
		if len(docs) != 1 {
			t.Errorf("Expected 1 document, got %d", len(docs))
		}
	})

	t.Run("Missing Files Handling", func(t *testing.T) {
		configMissing := config
		configMissing.DatasetPaths = []string{"nonexistent.txt", testFile}

		loader, err := NewDatasetLoader(configMissing)
		// Should succeed with at least one valid file
		if err != nil {
			t.Fatalf("Loader should handle missing files gracefully: %v", err)
		}

		docs := loader.GetDocuments()
		if len(docs) != 1 {
			t.Errorf("Expected 1 document from valid file, got %d", len(docs))
		}
	})

	t.Run("Word Similarity", func(t *testing.T) {
		loader, _ := NewDatasetLoader(config)
		
		// Test identical words
		sim := loader.ComputeSimilarity("hello", "hello")
		if sim <= 0.5 {
			t.Errorf("Identical words should have high similarity, got %f", sim)
		}

		// Test different words
		sim = loader.ComputeSimilarity("hello", "xyz123")
		if sim > 0.5 {
			t.Errorf("Different words should have low similarity, got %f", sim)
		}
	})

	t.Run("Large File Protection", func(t *testing.T) {
		largeFile := "large_test.txt"
		// Create a file larger than 10MB
		largeContent := strings.Repeat("test content ", 1000000)
		err := os.WriteFile(largeFile, []byte(largeContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create large test file: %v", err)
		}
		defer os.Remove(largeFile)

		configLarge := config
		configLarge.DatasetPaths = []string{largeFile}

		loader, err := NewDatasetLoader(configLarge)
		// Should handle large files gracefully
		if err == nil {
			t.Log("Large file handled successfully")
		} else {
			t.Logf("Large file rejected as expected: %v", err)
		}

		// Even if loader is created, it should work
		if loader != nil {
			vocab := loader.GetVocabulary()
			t.Logf("Vocabulary size: %d", len(vocab))
		}
	})
}

// TestLiquidStateBrain tests the liquid state brain functionality
func TestLiquidStateBrain(t *testing.T) {
	config := DefaultConfig()
	config.Resources.MaxNeurons = 1000
	config.Resources.MaxGoroutines = 50

	t.Run("Basic Creation", func(t *testing.T) {
		brain := NewLiquidStateBrainWithConfig(5, config)
		if brain == nil {
			t.Fatal("Failed to create brain")
		}
		defer brain.Cleanup()

		if brain.dimensions.X != 5 || brain.dimensions.Y != 5 || brain.dimensions.Z < 1 {
			t.Errorf("Unexpected brain dimensions: %+v", brain.dimensions)
		}
	})

	t.Run("Resource Limits", func(t *testing.T) {
		limitedConfig := config
		limitedConfig.Resources.MaxNeurons = 10

		// Try to create oversized brain
		brain := NewLiquidStateBrainWithConfig(50, limitedConfig)
		if brain == nil {
			t.Log("Oversized brain correctly rejected")
		} else {
			t.Log("Brain size was adjusted within limits")
			brain.Cleanup()
		}
	})

	t.Run("Invalid Size Handling", func(t *testing.T) {
		brain := NewLiquidStateBrainWithConfig(0, config)
		if brain != nil {
			t.Error("Brain with size 0 should be rejected")
			brain.Cleanup()
		}

		brain = NewLiquidStateBrainWithConfig(-1, config)
		if brain != nil {
			t.Error("Brain with negative size should be rejected")
			brain.Cleanup()
		}
	})

	t.Run("Processing and Cleanup", func(t *testing.T) {
		brain := NewLiquidStateBrainWithConfig(3, config)
		if brain == nil {
			t.Fatal("Failed to create brain")
		}

		// Test processing
		response := brain.Think("hello world")
		if response == "" {
			t.Error("Brain should generate some response")
		}

		// Test cleanup
		brain.Cleanup()
		// Should be safe to call cleanup multiple times
		brain.Cleanup()
	})

	t.Run("Concurrent Processing", func(t *testing.T) {
		brain := NewLiquidStateBrainWithConfig(4, config)
		if brain == nil {
			t.Fatal("Failed to create brain")
		}
		defer brain.Cleanup()

		// Test concurrent inputs
		responses := make(chan string, 3)
		for i := 0; i < 3; i++ {
			go func(idx int) {
				response := brain.Think("concurrent test")
				responses <- response
			}(i)
		}

		// Collect responses
		for i := 0; i < 3; i++ {
			select {
			case <-responses:
				// Got response
			case <-time.After(5 * time.Second):
				t.Error("Timeout waiting for concurrent response")
			}
		}
	})
}

// TestTransparentLLM tests the transparent LLM functionality
func TestTransparentLLM(t *testing.T) {
	config := DefaultConfig()
	config.Model.MaxConcepts = 100
	config.Resources.ChannelBufferSize = 10

	t.Run("Basic Creation", func(t *testing.T) {
		llm := NewTransparentLLMWithConfig(config)
		if llm == nil {
			t.Fatal("Failed to create TransparentLLM")
		}
		defer llm.Cleanup()

		if len(llm.concepts) == 0 {
			t.Error("LLM should have some concepts")
		}
	})

	t.Run("Understanding Process", func(t *testing.T) {
		llm := NewTransparentLLMWithConfig(config)
		if llm == nil {
			t.Fatal("Failed to create TransparentLLM")
		}
		defer llm.Cleanup()

		response, thoughtChan := llm.Understand("hello world test")
		if response == "" {
			t.Error("LLM should generate a response")
		}

		// Check thought process
		thoughtCount := 0
		timeout := time.After(3 * time.Second)
		for {
			select {
			case thought, ok := <-thoughtChan:
				if !ok {
					goto done
				}
				thoughtCount++
				if thought.stage == "" {
					t.Error("Thought should have a stage")
				}
			case <-timeout:
				goto done
			}
		}
		done:
		if thoughtCount == 0 {
			t.Error("Should have some thought traces")
		}
	})

	t.Run("Resource Constraints", func(t *testing.T) {
		constrainedConfig := config
		constrainedConfig.Model.MaxConcepts = 5
		constrainedConfig.Resources.ChannelBufferSize = 2

		llm := NewTransparentLLMWithConfig(constrainedConfig)
		if llm == nil {
			t.Fatal("Failed to create constrained LLM")
		}
		defer llm.Cleanup()

		// Should still work with constraints
		response, _ := llm.Understand("test")
		if response == "" {
			t.Error("Constrained LLM should still generate responses")
		}
	})

	t.Run("Nil Config Handling", func(t *testing.T) {
		llm := NewTransparentLLMWithConfig(nil)
		if llm == nil {
			t.Fatal("LLM should handle nil config with defaults")
		}
		defer llm.Cleanup()
	})
}

// TestConfig tests configuration loading and validation
func TestConfig(t *testing.T) {
	t.Run("Default Config", func(t *testing.T) {
		config := DefaultConfig()
		err := config.Validate()
		if err != nil {
			t.Errorf("Default config should be valid: %v", err)
		}
	})

	t.Run("Invalid Config Values", func(t *testing.T) {
		config := DefaultConfig()
		
		config.Model.EmbeddingDim = -1
		if config.Validate() == nil {
			t.Error("Negative embedding dim should be invalid")
		}

		config = DefaultConfig()
		config.Resources.MaxGoroutines = 0
		if config.Validate() == nil {
			t.Error("Zero max goroutines should be invalid")
		}

		config = DefaultConfig()
		config.Datasets.TestSplitRatio = 1.5
		if config.Validate() == nil {
			t.Error("Test split ratio > 1 should be invalid")
		}
	})

	t.Run("Config File Operations", func(t *testing.T) {
		testConfigFile := "test_config.json"
		defer os.Remove(testConfigFile)

		// Test saving
		config := DefaultConfig()
		err := SaveConfig(testConfigFile, config)
		if err != nil {
			t.Errorf("Failed to save config: %v", err)
		}

		// Test loading
		loadedConfig, err := LoadConfig(testConfigFile)
		if err != nil {
			t.Errorf("Failed to load config: %v", err)
		}

		if loadedConfig.Model.EmbeddingDim != config.Model.EmbeddingDim {
			t.Error("Loaded config doesn't match saved config")
		}
	})
}

// TestResponseGenerator tests the response generation functionality
func TestResponseGenerator(t *testing.T) {
	// Create minimal dataset for testing
	testFile := "test_responses.txt"
	testContent := "hello world how are you today this is a test artificial intelligence"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	config := TrainingConfig{
		DatasetPaths: []string{testFile},
		MaxVocabSize: 100,
		EmbeddingDim: 32,
		MinWordFreq:  1,
		MaxDocuments: 10,
	}

	loader, err := NewDatasetLoader(config)
	if err != nil {
		t.Fatalf("Failed to create dataset loader: %v", err)
	}

	t.Run("Basic Generation", func(t *testing.T) {
		generator := NewResponseGenerator(loader)
		if generator == nil {
			t.Fatal("Failed to create response generator")
		}

		response := generator.Generate("hello", []string{"world", "test"})
		if response == "" {
			t.Error("Generator should produce some response")
		}
	})

	t.Run("Different Input Types", func(t *testing.T) {
		generator := NewResponseGenerator(loader)

		inputs := []string{
			"hello world",
			"how are you",
			"artificial intelligence",
			"", // Empty input
		}

		for _, input := range inputs {
			response := generator.Generate(input, []string{"test"})
			// Should handle all inputs gracefully
			t.Logf("Input: '%s' -> Response: '%s'", input, response)
		}
	})
}

// TestUtils tests utility functions
func TestUtils(t *testing.T) {
	t.Run("Helper Functions", func(t *testing.T) {
		if min(5, 3) != 3 {
			t.Error("min function incorrect")
		}
		if max(5, 3) != 5 {
			t.Error("max function incorrect")
		}
	})

	t.Run("Graceful Shutdown", func(t *testing.T) {
		shutdown := NewGracefulShutdown(1 * time.Second)
		
		cleanupCalled := false
		shutdown.AddCleanup(func() {
			cleanupCalled = true
		})

		shutdown.Shutdown()
		if !cleanupCalled {
			t.Error("Cleanup function should have been called")
		}
	})

	t.Run("Resource Monitor", func(t *testing.T) {
		monitor := NewResourceMonitor(100, 100*time.Millisecond)
		monitor.Start()
		
		// Let it run briefly
		time.Sleep(200 * time.Millisecond)
		
		monitor.Stop()
		// Should stop gracefully
	})

	t.Run("Safe Goroutine", func(t *testing.T) {
		done := make(chan bool)
		
		SafeGoroutine("test", func() {
			// Simulate panic
			panic("test panic")
		})

		SafeGoroutine("test2", func() {
			done <- true
		})

		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Error("Safe goroutine should complete")
		}
	})
}

// TestErrorRecovery tests error handling and recovery mechanisms
func TestErrorRecovery(t *testing.T) {
	t.Run("Panic Recovery", func(t *testing.T) {
		config := DefaultConfig()
		config.Resources.MaxNeurons = 100

		// This should not crash the test even if there are panics
		brain := NewLiquidStateBrainWithConfig(3, config)
		if brain != nil {
			defer brain.Cleanup()
		}
	})

	t.Run("Context Cancellation", func(t *testing.T) {
		config := DefaultConfig()
		llm := NewTransparentLLMWithConfig(config)
		if llm == nil {
			t.Fatal("Failed to create LLM")
		}

		// Cancel context and cleanup
		llm.Cleanup()
		
		// Should handle gracefully
		t.Log("Context cancellation handled")
	})

	t.Run("Channel Overflow", func(t *testing.T) {
		config := DefaultConfig()
		config.Resources.ChannelBufferSize = 1 // Very small buffer

		brain := NewLiquidStateBrainWithConfig(2, config)
		if brain != nil {
			defer brain.Cleanup()

			// Try to overload with rapid inputs
			for i := 0; i < 10; i++ {
				go brain.Think("overload test")
			}

			time.Sleep(100 * time.Millisecond)
			t.Log("Channel overflow handled gracefully")
		}
	})
}

// TestIntegration tests the integration between components
func TestIntegration(t *testing.T) {
	t.Run("Full Pipeline", func(t *testing.T) {
		// Create test dataset
		testFile := "integration_test.txt"
		testContent := `
		Hello world, this is a test.
		Artificial intelligence and machine learning.
		Natural language processing is fascinating.
		The system should handle this gracefully.
		`
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		defer os.Remove(testFile)

		// Create config with test dataset
		config := DefaultConfig()
		config.Training.DatasetPaths = []string{testFile}
		config.Training.MaxDocuments = 1
		config.Resources.MaxNeurons = 500
		config.Model.MaxConcepts = 50

		// Test TransparentLLM
		llm := NewTransparentLLMWithConfig(config)
		if llm == nil {
			t.Fatal("Failed to create LLM")
		}
		defer llm.Cleanup()

		response, _ := llm.Understand("hello artificial intelligence")
		if response == "" {
			t.Error("LLM should generate response")
		}

		// Test LiquidStateBrain
		brain := NewLiquidStateBrainWithConfig(4, config)
		if brain == nil {
			t.Fatal("Failed to create brain")
		}
		defer brain.Cleanup()

		response = brain.Think("natural language processing")
		if response == "" {
			t.Error("Brain should generate response")
		}
	})

	t.Run("Stress Test", func(t *testing.T) {
		config := DefaultConfig()
		config.Resources.MaxNeurons = 200
		config.Resources.MaxGoroutines = 20
		config.Model.MaxConcepts = 20

		// Create multiple components
		llm := NewTransparentLLMWithConfig(config)
		brain := NewLiquidStateBrainWithConfig(3, config)

		if llm != nil && brain != nil {
			defer llm.Cleanup()
			defer brain.Cleanup()

			// Run concurrent operations
			for i := 0; i < 5; i++ {
				go func() {
					llm.Understand("stress test")
				}()
				go func() {
					brain.Think("stress test")
				}()
			}

			time.Sleep(500 * time.Millisecond)
			t.Log("Stress test completed")
		}
	})
}

// BenchmarkLiquidBrain benchmarks the liquid brain performance
func BenchmarkLiquidBrain(b *testing.B) {
	config := DefaultConfig()
	config.Resources.MaxNeurons = 1000
	brain := NewLiquidStateBrainWithConfig(5, config)
	if brain == nil {
		b.Fatal("Failed to create brain")
	}
	defer brain.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		brain.Think("benchmark test")
	}
}

// BenchmarkTransparentLLM benchmarks the transparent LLM performance
func BenchmarkTransparentLLM(b *testing.B) {
	config := DefaultConfig()
	config.Model.MaxConcepts = 100
	llm := NewTransparentLLMWithConfig(config)
	if llm == nil {
		b.Fatal("Failed to create LLM")
	}
	defer llm.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		llm.Understand("benchmark test")
	}
}