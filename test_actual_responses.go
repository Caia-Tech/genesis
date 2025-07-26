package main

import (
	"fmt"
	"strings"
)

func TestActualResponses() {
	fmt.Println("=== Testing What Genesis ACTUALLY Does ===")
	fmt.Println()
	
	// Test 1: TransparentLLM
	fmt.Println("1. TransparentLLM responses:")
	config := DefaultConfig()
	llm := NewTransparentLLMWithConfig(config)
	defer llm.Cleanup()
	
	testInputs := []string{
		"hello",
		"what is 2+2",
		"explain quantum physics",
		"I love you",
		"help me code",
		"the cat sat on the mat",
		"why is the sky blue",
		"recursive function",
	}
	
	for _, input := range testInputs {
		fmt.Printf("\nInput: '%s'\n", input)
		response, thoughtChan := llm.Understand(input)
		
		// Drain thoughts
		thoughts := []string{}
		for thought := range thoughtChan {
			if thought.stage == "understanding" && thought.insight != "" {
				thoughts = append(thoughts, thought.insight)
			}
		}
		
		fmt.Printf("Response: '%s'\n", response)
		fmt.Printf("Primary thought: %v\n", thoughts)
		fmt.Printf("Response length: %d words\n", len(strings.Fields(response)))
	}
	
	// Test 2: LiquidStateBrain
	fmt.Println("\n\n2. LiquidStateBrain responses:")
	brain := NewLiquidStateBrainWithConfig(1000, config)
	defer brain.Cleanup()
	
	for _, input := range testInputs {
		fmt.Printf("\nInput: '%s'\n", input)
		response := brain.Think(input)
		fmt.Printf("Response: '%s'\n", response)
		fmt.Printf("Response length: %d words\n", len(strings.Fields(response)))
		
		// Check if response varies
		response2 := brain.Think(input)
		if response != response2 {
			fmt.Printf("Second response: '%s' (VARIES!)\n", response2)
		} else {
			fmt.Printf("Second response: SAME\n")
		}
	}
	
	// Test 3: What happens with more neurons?
	fmt.Println("\n\n3. Scaling test - does more neurons = different behavior?")
	
	sizes := []int{100, 1000, 10000}
	testInput := "explain consciousness"
	
	for _, size := range sizes {
		fmt.Printf("\nBrain size: %d neurons\n", size)
		testBrain := NewLiquidStateBrainWithConfig(size, config)
		
		response := testBrain.Think(testInput)
		fmt.Printf("Response: '%s'\n", response)
		fmt.Printf("Response length: %d words\n", len(strings.Fields(response)))
		
		testBrain.Cleanup()
	}
	
	// Test 4: Check actual wave patterns
	fmt.Println("\n\n4. Wave pattern analysis:")
	brain2 := NewLiquidStateBrainWithConfig(100, config)
	defer brain2.Cleanup()
	
	// Count active waves
	waveCount := 0
	go func() {
		for range brain2.wavePatterns {
			waveCount++
		}
	}()
	
	brain2.Think("test wave propagation")
	fmt.Printf("Wave patterns generated: %d\n", waveCount)
}