package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// DemoMain runs the interactive transparent AI demo
func DemoMain() {
	fmt.Println("\n🤖 Welcome to Genesis Transparent AI Demo")
	fmt.Println("Watch as the AI shows its thinking process!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("Type 'quit' to exit")
	fmt.Println()

	// Load configuration
	config := DefaultConfig()
	llm := NewTransparentLLMWithConfig(config)
	defer llm.Cleanup()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\n💭 You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "quit" || input == "exit" {
			fmt.Println("\n👋 Goodbye!")
			break
		}

		if input == "" {
			continue
		}

		// Process with transparent LLM
		start := time.Now()
		response, thoughtStream := llm.Understand(input)

		// Show thinking process
		fmt.Println("\n🧠 AI Thinking Process:")
		fmt.Println(strings.Repeat("-", 40))

		thoughtCount := 0
		for thought := range thoughtStream {
			thoughtCount++
			// Thoughts are already visualized by the model
			_ = thought
		}

		duration := time.Since(start)
		
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("\n🤖 AI: %s\n", response)
		fmt.Printf("\n⏱️  Response generated in %v with %d thought steps\n", duration, thoughtCount)
	}
}

// RunAutoDemo runs an automated demonstration
func RunAutoDemo() {
	fmt.Println("\n🚀 Genesis AI Automated Demo")
	fmt.Println(strings.Repeat("=", 50))
	
	// Test cases for demonstration
	testInputs := []struct {
		input       string
		description string
	}{
		{
			"I'm stuck on this code and feeling frustrated",
			"Emotional understanding test",
		},
		{
			"Can you help me debug this error?",
			"Technical assistance test",
		},
		{
			"What's the meaning of this pattern?",
			"Abstract reasoning test",
		},
		{
			"Hello, how are you?",
			"Social interaction test",
		},
	}

	// Create both models for comparison
	config := DefaultConfig()
	
	fmt.Println("\n1️⃣  Testing Transparent LLM...")
	transparentLLM := NewTransparentLLMWithConfig(config)
	defer transparentLLM.Cleanup()

	for _, test := range testInputs {
		fmt.Printf("\n📝 Test: %s\n", test.description)
		fmt.Printf("Input: \"%s\"\n", test.input)
		
		start := time.Now()
		response, thoughtStream := transparentLLM.Understand(test.input)
		
		// Drain thought stream
		thoughtSteps := 0
		for range thoughtStream {
			thoughtSteps++
		}
		
		fmt.Printf("Response: %s\n", response)
		fmt.Printf("Time: %v | Thought steps: %d\n", time.Since(start), thoughtSteps)
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("\n2️⃣  Testing Liquid State Brain...")
	
	liquidBrain := NewLiquidStateBrainWithConfig(20, config) // Smaller brain for demo
	defer liquidBrain.Cleanup()
	
	time.Sleep(1 * time.Second) // Let it initialize

	for _, test := range testInputs {
		fmt.Printf("\n📝 Test: %s\n", test.description)
		fmt.Printf("Input: \"%s\"\n", test.input)
		
		start := time.Now()
		response := liquidBrain.Think(test.input)
		
		fmt.Printf("Response: %s\n", response)
		fmt.Printf("Time: %v\n", time.Since(start))
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ Demo complete!")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("- Transparent reasoning process")
	fmt.Println("- Parallel neural processing")
	fmt.Println("- Dataset integration")
	fmt.Println("- Resource management")
	fmt.Println("- Concurrent safety")
	
	fmt.Println("\n📊 Performance Summary:")
	fmt.Printf("- Transparent LLM: Fast responses with visible reasoning\n")
	fmt.Printf("- Liquid Brain: Complex 3D reservoir dynamics\n")
	
	fmt.Println("\n🎯 Next Steps:")
	fmt.Println("1. Run with 'train' flag to train on your datasets")
	fmt.Println("2. Modify config.json to tune parameters")
	fmt.Println("3. Add more datasets to the datasets/ directory")
}