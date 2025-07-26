package main

import (
	"fmt"
	"log"
)

func TestSimple() {
	fmt.Println("=== Simple Genesis LLM Test ===")
	
	// Test 1: Configuration
	fmt.Println("\n1. Testing configuration...")
	config := DefaultConfig()
	fmt.Printf("   Model type: %s\n", config.Model.Type)
	fmt.Printf("   Max vocab size: %d\n", config.Training.MaxVocabSize)
	fmt.Printf("   Dataset paths: %v\n", config.Training.DatasetPaths)
	
	// Test 2: Dataset Loading
	fmt.Println("\n2. Testing dataset loader...")
	loader, err := NewDatasetLoader(config.Training)
	if err != nil {
		log.Printf("   ERROR loading dataset: %v", err)
		return
	}
	
	vocab := loader.GetVocabulary()
	fmt.Printf("   Loaded vocabulary: %d words\n", len(vocab))
	if len(vocab) > 0 {
		fmt.Printf("   Sample words: %v\n", vocab[:min(10, len(vocab))])
	}
	
	docs := loader.GetDocuments()
	fmt.Printf("   Loaded documents: %d\n", len(docs))
	
	// Test 3: TransparentLLM
	fmt.Println("\n3. Testing TransparentLLM...")
	llm := NewTransparentLLMWithConfig(config)
	defer llm.Cleanup()
	
	if len(llm.concepts) > 0 {
		fmt.Printf("   Initialized with %d concepts\n", len(llm.concepts))
		
		// Test a simple input
		fmt.Println("   Testing input: 'hello world'")
		response, thoughtChan := llm.Understand("hello world")
		
		// Drain thought channel
		thoughtCount := 0
		for range thoughtChan {
			thoughtCount++
		}
		
		fmt.Printf("   Response: %s\n", response)
		fmt.Printf("   Thought steps: %d\n", thoughtCount)
	} else {
		fmt.Println("   WARNING: No concepts loaded")
	}
	
	// Test 4: Basic LiquidStateBrain (smaller size)
	fmt.Println("\n4. Testing LiquidStateBrain...")
	brain := NewLiquidStateBrainWithConfig(10, config) // Very small brain
	defer brain.Cleanup()
	
	fmt.Printf("   Created brain: %dx%dx%d\n", brain.dimensions.X, brain.dimensions.Y, brain.dimensions.Z)
	
	fmt.Println("\n=== Test Complete ===")
}