package main

import (
	"fmt"
	"math/rand"
	"strings"
)

func RunXORExperiment() {
	fmt.Println("=== XOR Gate Discovery Experiment ===")
	fmt.Println("Evolving a circuit to implement XOR logic...")
	
	testCases := []TestCase{
		{[]bool{false, false}, false},
		{[]bool{false, true}, true},
		{[]bool{true, false}, true},
		{[]bool{true, true}, false},
	}
	
	evolution := NewEvolution(50, testCases)
	logger := NewEvolutionLogger(evolution)
	
	for gen := 0; gen < 100; gen++ {
		evolution.RunGeneration()
		logger.LogGeneration(gen)
		
		if evolution.bestFitness >= 1.0 {
			fmt.Printf("\nPerfect solution found at generation %d!\n", gen)
			break
		}
	}
	
	logger.PrintFinalReport()
}

func RunParityExperiment() {
	fmt.Println("\n=== 3-bit Parity Checker Discovery ===")
	fmt.Println("Evolving a circuit to check if number of true inputs is odd...")
	
	testCases := []TestCase{
		{[]bool{false, false, false}, false},
		{[]bool{false, false, true}, true},
		{[]bool{false, true, false}, true},
		{[]bool{false, true, true}, false},
		{[]bool{true, false, false}, true},
		{[]bool{true, false, true}, false},
		{[]bool{true, true, false}, false},
		{[]bool{true, true, true}, true},
	}
	
	evolution := NewEvolution(100, testCases)
	logger := NewEvolutionLogger(evolution)
	
	for gen := 0; gen < 200; gen++ {
		evolution.RunGeneration()
		logger.LogGeneration(gen)
		
		if evolution.bestFitness >= 1.0 {
			fmt.Printf("\nPerfect solution found at generation %d!\n", gen)
			break
		}
	}
	
	logger.PrintFinalReport()
}

func RunMajorityExperiment() {
	fmt.Println("\n=== Majority Vote Circuit Discovery ===")
	fmt.Println("Evolving a circuit to output true if majority of inputs are true...")
	
	testCases := []TestCase{
		{[]bool{false, false, false}, false},
		{[]bool{false, false, true}, false},
		{[]bool{false, true, false}, false},
		{[]bool{false, true, true}, true},
		{[]bool{true, false, false}, false},
		{[]bool{true, false, true}, true},
		{[]bool{true, true, false}, true},
		{[]bool{true, true, true}, true},
	}
	
	evolution := NewEvolution(100, testCases)
	logger := NewEvolutionLogger(evolution)
	
	for gen := 0; gen < 200; gen++ {
		evolution.RunGeneration()
		logger.LogGeneration(gen)
		
		if evolution.bestFitness >= 1.0 {
			fmt.Printf("\nPerfect solution found at generation %d!\n", gen)
			break
		}
	}
	
	logger.PrintFinalReport()
}

func RunSelfDiscoveryExperiment() {
	fmt.Println("\n=== Self-Discovery Experiment ===")
	fmt.Println("Circuit discovers its own function from random test cases...")
	
	numInputs := 2
	numTests := 8
	
	mysteryFunction := func(inputs []bool) bool {
		if len(inputs) >= 2 {
			return (inputs[0] || inputs[1]) && !(inputs[0] && inputs[1])
		}
		return false
	}
	
	testCases := make([]TestCase, numTests)
	for i := 0; i < numTests; i++ {
		inputs := make([]bool, numInputs)
		for j := 0; j < numInputs; j++ {
			inputs[j] = rand.Float32() < 0.5
		}
		testCases[i] = TestCase{
			Input:    inputs,
			Expected: mysteryFunction(inputs),
		}
	}
	
	fmt.Println("Mystery function test cases:")
	for _, tc := range testCases {
		fmt.Printf("  %v -> %v\n", tc.Input, tc.Expected)
	}
	
	evolution := NewEvolution(150, testCases)
	logger := NewEvolutionLogger(evolution)
	
	for gen := 0; gen < 300; gen++ {
		evolution.RunGeneration()
		
		if gen%20 == 0 {
			logger.LogGeneration(gen)
		}
		
		if evolution.bestFitness >= 1.0 {
			fmt.Printf("\nPerfect match found at generation %d!\n", gen)
			break
		}
	}
	
	if evolution.bestFitness >= 1.0 {
		fmt.Println("\nVerifying on complete truth table:")
		allInputs := [][]bool{
			{false, false},
			{false, true},
			{true, false},
			{true, true},
		}
		
		allCorrect := true
		for _, inputs := range allInputs {
			expected := mysteryFunction(inputs)
			if len(evolution.bestCircuit.gates) > 0 {
				actual := evolution.bestCircuit.gates[len(evolution.bestCircuit.gates)-1].Process(inputs)
				match := actual == expected
				if !match {
					allCorrect = false
				}
				fmt.Printf("  %v -> %v (expected %v) %s\n", 
					inputs, actual, expected, 
					map[bool]string{true: "✓", false: "✗"}[match])
			}
		}
		
		if allCorrect {
			fmt.Println("\nThe circuit successfully discovered: XOR function!")
		}
	}
	
	logger.PrintFinalReport()
}

func RunAllExperiments() {
	// Go 1.20+ uses automatic seeding
	
	experiments := []func(){
		RunXORExperiment,
		RunParityExperiment,
		RunMajorityExperiment,
		RunSelfDiscoveryExperiment,
	}
	
	for _, exp := range experiments {
		exp()
		fmt.Println("\n" + strings.Repeat("=", 50) + "\n")
	}
}