package main

import (
	"fmt"
	"strings"
)

type CircuitVisualizer struct {
	circuit *EvolvingCircuit
}

func NewCircuitVisualizer(circuit *EvolvingCircuit) *CircuitVisualizer {
	return &CircuitVisualizer{circuit: circuit}
}

func (cv *CircuitVisualizer) PrintCircuit() {
	cv.circuit.mu.RLock()
	defer cv.circuit.mu.RUnlock()
	
	fmt.Printf("\n=== Circuit Generation %d ===\n", cv.circuit.generation)
	fmt.Printf("Fitness: %.4f\n", cv.circuit.fitness)
	fmt.Printf("Gates: %d\n", len(cv.circuit.gates))
	
	connections := make(map[string][]string)
	gateTypes := make(map[string]string)
	
	for i, gate := range cv.circuit.gates {
		gateID := fmt.Sprintf("g%d", i)
		
		switch g := gate.(type) {
		case *BaseGate:
			gateTypes[gateID] = "Base"
			g.mu.RLock()
			for j, input := range g.inputs {
				for k, searchGate := range cv.circuit.gates {
					if searchGate == input {
						connections[gateID] = append(connections[gateID], fmt.Sprintf("g%d", k))
						break
					}
				}
				if j >= 5 {
					connections[gateID] = append(connections[gateID], "...")
					break
				}
			}
			g.mu.RUnlock()
		case *AdaptiveGate:
			gateTypes[gateID] = fmt.Sprintf("Adaptive(mem:%d)", g.memorySize)
			g.mu.RLock()
			for j, input := range g.inputs {
				for k, searchGate := range cv.circuit.gates {
					if searchGate == input {
						connections[gateID] = append(connections[gateID], fmt.Sprintf("g%d", k))
						break
					}
				}
				if j >= 5 {
					connections[gateID] = append(connections[gateID], "...")
					break
				}
			}
			g.mu.RUnlock()
		}
	}
	
	fmt.Println("\nGate Connections:")
	for i := 0; i < len(cv.circuit.gates); i++ {
		gateID := fmt.Sprintf("g%d", i)
		gateType := gateTypes[gateID]
		inputs := connections[gateID]
		
		if len(inputs) > 0 {
			fmt.Printf("  %s [%s] <- %s\n", gateID, gateType, strings.Join(inputs, ", "))
		} else {
			fmt.Printf("  %s [%s] <- (no inputs)\n", gateID, gateType)
		}
	}
	
	if len(cv.circuit.gates) > 0 {
		fmt.Printf("\nOutput gate: g%d\n", len(cv.circuit.gates)-1)
	}
}

type EvolutionLogger struct {
	evolution    *Evolution
	logFrequency int
	history      []float64
}

func NewEvolutionLogger(evolution *Evolution) *EvolutionLogger {
	return &EvolutionLogger{
		evolution:    evolution,
		logFrequency: 10,
		history:      make([]float64, 0),
	}
}

func (el *EvolutionLogger) LogGeneration(generation int) {
	el.history = append(el.history, el.evolution.bestFitness)
	
	if generation%el.logFrequency == 0 || generation == 0 {
		fmt.Printf("\nGeneration %d:\n", generation)
		fmt.Printf("  Best Fitness: %.4f\n", el.evolution.bestFitness)
		
		if el.evolution.bestCircuit != nil {
			fmt.Printf("  Circuit Size: %d gates\n", len(el.evolution.bestCircuit.gates))
			if len(el.evolution.bestCircuit.gates) > 0 {
				fmt.Printf("  Complexity: %d\n", el.evolution.bestCircuit.gates[len(el.evolution.bestCircuit.gates)-1].Complexity())
			}
			
			correct := 0
			for _, tc := range el.evolution.testCases {
				if len(el.evolution.bestCircuit.gates) > 0 {
					output := el.evolution.bestCircuit.gates[len(el.evolution.bestCircuit.gates)-1].Process(tc.Input)
					if output == tc.Expected {
						correct++
					}
				}
			}
			fmt.Printf("  Correct: %d/%d\n", correct, len(el.evolution.testCases))
		}
		
		el.PrintFitnessHistory()
	}
}

func (el *EvolutionLogger) PrintFitnessHistory() {
	if len(el.history) < 2 {
		return
	}
	
	fmt.Print("  Progress: ")
	
	maxFit := 0.0
	minFit := 1.0
	for _, fit := range el.history {
		if fit > maxFit {
			maxFit = fit
		}
		if fit < minFit {
			minFit = fit
		}
	}
	
	sparkline := ""
	for _, fit := range el.history[max(0, len(el.history)-20):] {
		normalized := (fit - minFit) / (maxFit - minFit + 0.0001)
		if normalized < 0.2 {
			sparkline += "▁"
		} else if normalized < 0.4 {
			sparkline += "▃"
		} else if normalized < 0.6 {
			sparkline += "▅"
		} else if normalized < 0.8 {
			sparkline += "▇"
		} else {
			sparkline += "█"
		}
	}
	
	fmt.Println(sparkline)
}

// max function already defined in other files

func (el *EvolutionLogger) PrintFinalReport() {
	fmt.Println("\n=== Evolution Complete ===")
	
	if el.evolution.bestCircuit != nil {
		visualizer := NewCircuitVisualizer(el.evolution.bestCircuit)
		visualizer.PrintCircuit()
		
		fmt.Println("\nTest Results:")
		allCorrect := true
		for _, tc := range el.evolution.testCases {
			if len(el.evolution.bestCircuit.gates) > 0 {
				output := el.evolution.bestCircuit.gates[len(el.evolution.bestCircuit.gates)-1].Process(tc.Input)
				correct := output == tc.Expected
				status := "✓"
				if !correct {
					status = "✗"
					allCorrect = false
				}
				fmt.Printf("  %s Input: %v -> Output: %v (Expected: %v)\n", 
					status, tc.Input, output, tc.Expected)
			}
		}
		
		if allCorrect {
			fmt.Println("\n🎉 Perfect solution found!")
		}
	}
}