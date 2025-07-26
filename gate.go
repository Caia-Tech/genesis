package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
)

type Signal interface{}

type Gate interface {
	ID() string
	Connect(input Gate)
	Disconnect(input Gate)
	Process(signal Signal) Signal
	Mutate() Gate
	Clone() Gate
	Complexity() int
}

type BaseGate struct {
	id       string
	inputs   []Gate
	output   chan Signal
	function func([]Signal) Signal
	mu       sync.RWMutex
}

func NewBaseGate(id string, fn func([]Signal) Signal) *BaseGate {
	return &BaseGate{
		id:       id,
		inputs:   make([]Gate, 0),
		output:   make(chan Signal, 100),
		function: fn,
	}
}

func (g *BaseGate) ID() string { return g.id }

func (g *BaseGate) Connect(input Gate) {
	g.mu.Lock()
	g.inputs = append(g.inputs, input)
	g.mu.Unlock()
}

func (g *BaseGate) Disconnect(input Gate) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	for i, in := range g.inputs {
		if in.ID() == input.ID() {
			g.inputs = append(g.inputs[:i], g.inputs[i+1:]...)
			break
		}
	}
}

func (g *BaseGate) Process(signal Signal) Signal {
	return g.ProcessWithVisited(signal, make(map[string]bool))
}

func (g *BaseGate) ProcessWithVisited(signal Signal, visited map[string]bool) Signal {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	if visited[g.id] {
		return false
	}
	visited[g.id] = true
	
	if len(g.inputs) == 0 {
		return g.function([]Signal{signal})
	}
	
	inputSignals := make([]Signal, len(g.inputs))
	for i, input := range g.inputs {
		if bg, ok := input.(*BaseGate); ok {
			inputSignals[i] = bg.ProcessWithVisited(signal, visited)
		} else if ag, ok := input.(*AdaptiveGate); ok {
			inputSignals[i] = ag.ProcessWithVisited(signal, visited)
		} else {
			inputSignals[i] = signal
		}
	}
	
	return g.function(inputSignals)
}

func (g *BaseGate) Complexity() int {
	visited := make(map[string]bool)
	return g.ComplexityWithVisited(visited)
}

func (g *BaseGate) ComplexityWithVisited(visited map[string]bool) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	if visited[g.id] {
		return 0
	}
	visited[g.id] = true
	
	complexity := 1
	for _, input := range g.inputs {
		if bg, ok := input.(*BaseGate); ok {
			complexity += bg.ComplexityWithVisited(visited)
		} else if ag, ok := input.(*AdaptiveGate); ok {
			complexity += ag.ComplexityWithVisited(visited)
		}
	}
	return complexity
}

func (g *BaseGate) Mutate() Gate {
	mutations := []func(){
		func() {
			if len(g.inputs) > 0 && rand.Float32() < 0.3 {
				idx := rand.Intn(len(g.inputs))
				g.inputs = append(g.inputs[:idx], g.inputs[idx+1:]...)
			}
		},
		func() {
			if rand.Float32() < 0.2 {
				g.function = RandomFunction()
			}
		},
	}
	
	mutation := mutations[rand.Intn(len(mutations))]
	mutation()
	
	return g
}

func (g *BaseGate) Clone() Gate {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	clone := &BaseGate{
		id:       fmt.Sprintf("%s_clone_%d", g.id, rand.Int()),
		inputs:   make([]Gate, len(g.inputs)),
		output:   make(chan Signal, 100),
		function: g.function,
	}
	copy(clone.inputs, g.inputs)
	return clone
}

func extractBools(signals []Signal) []bool {
	bools := []bool{}
	for _, s := range signals {
		switch v := s.(type) {
		case bool:
			bools = append(bools, v)
		case []bool:
			bools = append(bools, v...)
		}
	}
	return bools
}

func RandomFunction() func([]Signal) Signal {
	functions := []func([]Signal) Signal{
		func(inputs []Signal) Signal {
			if len(inputs) == 0 {
				return false
			}
			
			allBools := extractBools(inputs)
			if len(allBools) == 0 {
				return false
			}
			
			for _, b := range allBools {
				if b {
					return true
				}
			}
			return false
		},
		func(inputs []Signal) Signal {
			if len(inputs) == 0 {
				return true
			}
			
			allBools := extractBools(inputs)
			if len(allBools) == 0 {
				return true
			}
			
			for _, b := range allBools {
				if !b {
					return false
				}
			}
			return true
		},
		func(inputs []Signal) Signal {
			allBools := extractBools(inputs)
			if len(allBools) == 0 {
				return false
			}
			return !allBools[0]
		},
		func(inputs []Signal) Signal {
			allBools := extractBools(inputs)
			count := 0
			for _, b := range allBools {
				if b {
					count++
				}
			}
			return count%2 == 1
		},
		func(inputs []Signal) Signal {
			allBools := extractBools(inputs)
			if len(allBools) < 2 {
				return false
			}
			return allBools[0] != allBools[1]
		},
		func(inputs []Signal) Signal {
			allBools := extractBools(inputs)
			if len(allBools) < 2 {
				return false
			}
			return allBools[0] && !allBools[1]
		},
		func(inputs []Signal) Signal {
			allBools := extractBools(inputs)
			if len(allBools) < 2 {
				return false
			}
			return !allBools[0] && allBools[1]
		},
	}
	
	return functions[rand.Intn(len(functions))]
}

type AdaptiveGate struct {
	*BaseGate
	memory     []Signal
	memorySize int
}

func NewAdaptiveGate(id string) *AdaptiveGate {
	ag := &AdaptiveGate{
		BaseGate:   NewBaseGate(id, nil),
		memory:     make([]Signal, 0),
		memorySize: 5,
	}
	
	ag.function = func(inputs []Signal) Signal {
		ag.memory = append(ag.memory, inputs...)
		if len(ag.memory) > ag.memorySize {
			ag.memory = ag.memory[len(ag.memory)-ag.memorySize:]
		}
		
		trueCount := 0
		for _, mem := range ag.memory {
			if b, ok := mem.(bool); ok && b {
				trueCount++
			}
		}
		
		return float64(trueCount) > float64(len(ag.memory))*0.6
	}
	
	return ag
}

func (ag *AdaptiveGate) ProcessWithVisited(signal Signal, visited map[string]bool) Signal {
	ag.mu.RLock()
	defer ag.mu.RUnlock()
	
	if visited[ag.id] {
		return false
	}
	visited[ag.id] = true
	
	if len(ag.inputs) == 0 {
		return ag.function([]Signal{signal})
	}
	
	inputSignals := make([]Signal, len(ag.inputs))
	for i, input := range ag.inputs {
		if bg, ok := input.(*BaseGate); ok {
			inputSignals[i] = bg.ProcessWithVisited(signal, visited)
		} else if ag2, ok := input.(*AdaptiveGate); ok {
			inputSignals[i] = ag2.ProcessWithVisited(signal, visited)
		} else {
			inputSignals[i] = signal
		}
	}
	
	result := ag.function(inputSignals)
	
	ag.memory = append(ag.memory, result)
	if len(ag.memory) > ag.memorySize {
		ag.memory = ag.memory[len(ag.memory)-ag.memorySize:]
	}
	
	return result
}

func (ag *AdaptiveGate) ComplexityWithVisited(visited map[string]bool) int {
	return ag.BaseGate.ComplexityWithVisited(visited) + 1
}

func (ag *AdaptiveGate) Mutate() Gate {
	ag.BaseGate.Mutate()
	
	if rand.Float32() < 0.3 {
		ag.memorySize = ag.memorySize + rand.Intn(5) - 2
		if ag.memorySize < 1 {
			ag.memorySize = 1
		}
		if ag.memorySize > 20 {
			ag.memorySize = 20
		}
	}
	
	return ag
}

type EvolvingCircuit struct {
	gates      []Gate
	fitness    float64
	generation int
	mu         sync.RWMutex
}

func NewEvolvingCircuit(initialGates int) *EvolvingCircuit {
	ec := &EvolvingCircuit{
		gates:      make([]Gate, initialGates),
		generation: 0,
	}
	
	for i := 0; i < initialGates; i++ {
		if rand.Float32() < 0.5 {
			ec.gates[i] = NewBaseGate(fmt.Sprintf("gate_%d", i), RandomFunction())
		} else {
			ec.gates[i] = NewAdaptiveGate(fmt.Sprintf("adaptive_%d", i))
		}
	}
	
	for i := 0; i < initialGates*2; i++ {
		from := rand.Intn(len(ec.gates))
		to := rand.Intn(len(ec.gates))
		if from != to {
			ec.gates[to].Connect(ec.gates[from])
		}
	}
	
	return ec
}

func (ec *EvolvingCircuit) Evaluate(testCases []TestCase) float64 {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	
	correct := 0
	total := len(testCases)
	
	for _, tc := range testCases {
		output := ec.gates[len(ec.gates)-1].Process(tc.Input)
		if output == tc.Expected {
			correct++
		}
	}
	
	complexity := 0
	for _, gate := range ec.gates {
		complexity += gate.Complexity()
	}
	
	complexityPenalty := float64(complexity) * 0.001
	
	ec.fitness = float64(correct)/float64(total) - complexityPenalty
	return ec.fitness
}

func (ec *EvolvingCircuit) Mutate() *EvolvingCircuit {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	mutated := &EvolvingCircuit{
		gates:      make([]Gate, len(ec.gates)),
		generation: ec.generation + 1,
	}
	
	for i, gate := range ec.gates {
		if rand.Float32() < 0.8 {
			mutated.gates[i] = gate.Clone()
		} else {
			mutated.gates[i] = gate.Clone().Mutate()
		}
	}
	
	if rand.Float32() < 0.3 && len(mutated.gates) < 20 {
		newGate := NewBaseGate(fmt.Sprintf("new_%d", ec.generation), RandomFunction())
		mutated.gates = append(mutated.gates, newGate)
		
		for i := 0; i < rand.Intn(3)+1; i++ {
			target := rand.Intn(len(mutated.gates))
			mutated.gates[target].Connect(newGate)
		}
	}
	
	if rand.Float32() < 0.1 && len(mutated.gates) > 3 {
		idx := rand.Intn(len(mutated.gates))
		mutated.gates = append(mutated.gates[:idx], mutated.gates[idx+1:]...)
	}
	
	for i := 0; i < rand.Intn(5); i++ {
		from := rand.Intn(len(mutated.gates))
		to := rand.Intn(len(mutated.gates))
		if from != to {
			if rand.Float32() < 0.5 {
				mutated.gates[to].Connect(mutated.gates[from])
			} else {
				mutated.gates[to].Disconnect(mutated.gates[from])
			}
		}
	}
	
	return mutated
}

type TestCase struct {
	Input    Signal
	Expected Signal
}

type Evolution struct {
	population   []*EvolvingCircuit
	testCases    []TestCase
	bestCircuit  *EvolvingCircuit
	bestFitness  float64
	logFrequency int
}

func NewEvolution(populationSize int, testCases []TestCase) *Evolution {
	e := &Evolution{
		population:   make([]*EvolvingCircuit, populationSize),
		testCases:    testCases,
		logFrequency: 10,
	}
	
	for i := 0; i < populationSize; i++ {
		e.population[i] = NewEvolvingCircuit(rand.Intn(5) + 3)
	}
	
	return e
}

func (e *Evolution) RunGeneration() {
	for _, circuit := range e.population {
		fitness := circuit.Evaluate(e.testCases)
		if fitness > e.bestFitness {
			e.bestFitness = fitness
			e.bestCircuit = circuit
		}
	}
	
	newPopulation := make([]*EvolvingCircuit, len(e.population))
	
	eliteCount := len(e.population) / 10
	for i := 0; i < eliteCount; i++ {
		newPopulation[i] = e.bestCircuit
	}
	
	for i := eliteCount; i < len(e.population); i++ {
		parent := e.selectParent()
		newPopulation[i] = parent.Mutate()
	}
	
	e.population = newPopulation
}

func (e *Evolution) selectParent() *EvolvingCircuit {
	tournament := make([]*EvolvingCircuit, 3)
	for i := 0; i < 3; i++ {
		tournament[i] = e.population[rand.Intn(len(e.population))]
	}
	
	best := tournament[0]
	bestFit := tournament[0].Evaluate(e.testCases)
	
	for i := 1; i < 3; i++ {
		fit := tournament[i].Evaluate(e.testCases)
		if fit > bestFit {
			best = tournament[i]
			bestFit = fit
		}
	}
	
	return best
}

func (e *Evolution) Run(generations int) {
	for gen := 0; gen < generations; gen++ {
		e.RunGeneration()
		
		if gen%e.logFrequency == 0 || gen == generations-1 {
			fmt.Printf("Generation %d: Best fitness = %.4f, Circuit size = %d\n", 
				gen, e.bestFitness, len(e.bestCircuit.gates))
		}
	}
}

func main() {
	// Check command line arguments
	if len(os.Args) > 1 && os.Args[1] == "train" {
		// Run training mode
		TrainMain()
		return
	}
	
	// Otherwise run demos
	fmt.Println("Genesis LLM - Choose a mode:")
	fmt.Println("1. Evolution experiments")
	fmt.Println("2. Interactive transparent AI demo")
	fmt.Println("3. Automated demo")
	fmt.Println("4. Training mode (use 'go run . train')")
	fmt.Println("5. Simple test")
	fmt.Println("6. Response generation test")
	fmt.Println("7. Orchestration demo")
	fmt.Println("8. Parallel orchestration demo")
	fmt.Println("9. Scaling behavior demo")
	fmt.Println("10. Test actual responses")
	fmt.Print("\nSelection (default=3): ")
	
	var selection string
	fmt.Scanln(&selection)
	
	switch selection {
	case "1":
		RunAllExperiments()
	case "2":
		DemoMain()
	case "4":
		TrainMain()
	case "5":
		TestSimple()
	case "6":
		TestResponses()
	case "7":
		DemoOrchestration()
	case "8":
		DemoParallelOrchestration()
	case "9":
		ShowScalingBehavior()
	case "10":
		TestActualResponses()
	default:
		RunAutoDemo()
	}
}