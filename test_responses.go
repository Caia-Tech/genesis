package main

import (
	"fmt"
	"strings"
)

func TestResponses() {
	fmt.Println("=== Testing LLM Response Generation ===")
	fmt.Println()
	
	// Load configuration
	config := DefaultConfig()
	config.Training.MinWordFreq = 1 // Lower threshold for testing
	
	// Test TransparentLLM
	fmt.Println("1. Testing TransparentLLM responses:")
	llm := NewTransparentLLMWithConfig(config)
	defer llm.Cleanup()
	
	testInputs := []string{
		"hello",
		"how are you",
		"tell me about code",
		"I need help",
	}
	
	for _, input := range testInputs {
		fmt.Printf("\n   Input: '%s'\n", input)
		response, thoughtChan := llm.Understand(input)
		
		// Drain thought channel
		thoughtCount := 0
		for range thoughtChan {
			thoughtCount++
		}
		
		fmt.Printf("   Response: '%s'\n", response)
		fmt.Printf("   (Generated %d words, %d thought steps)\n", 
			len(strings.Fields(response)), thoughtCount)
	}
	
	// Test LiquidStateBrain
	fmt.Println("\n\n2. Testing LiquidStateBrain responses:")
	brain := NewLiquidStateBrainWithConfig(10, config)
	defer brain.Cleanup()
	
	for _, input := range testInputs {
		fmt.Printf("\n   Input: '%s'\n", input)
		response := brain.Think(input)
		fmt.Printf("   Response: '%s'\n", response)
		fmt.Printf("   (Generated %d words)\n", len(strings.Fields(response)))
	}
	
	fmt.Println("\n=== Test Complete ===")
}