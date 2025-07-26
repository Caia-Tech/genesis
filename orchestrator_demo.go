package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// OrchestratorNeuron - A neuron that can call external services
type OrchestratorNeuron struct {
	*LiquidNeuron
	capability string
	endpoint   func(context.Context, string) (string, error)
}

// GenesisOrchestrator - Transparent AI orchestration layer
type GenesisOrchestrator struct {
	liquidBrain *LiquidStateBrain
	neurons     map[string]*OrchestratorNeuron
	decisions   chan Decision
	mu          sync.RWMutex
}

type Decision struct {
	Input     string
	Path      []string
	Reasoning string
	Output    string
	Timestamp time.Time
}

// Example external capabilities (in production, these would call real APIs)
func mockGPT4(ctx context.Context, prompt string) (string, error) {
	return fmt.Sprintf("[GPT-4 response to: %s]", prompt), nil
}

func mockClaude(ctx context.Context, prompt string) (string, error) {
	return fmt.Sprintf("[Claude response to: %s]", prompt), nil
}

func mockCalculator(ctx context.Context, expr string) (string, error) {
	return fmt.Sprintf("[Calculated: %s = 42]", expr), nil
}

func mockDatabase(ctx context.Context, query string) (string, error) {
	return fmt.Sprintf("[DB result for: %s]", query), nil
}

// NewGenesisOrchestrator creates a transparent orchestration system
func NewGenesisOrchestrator(size int) *GenesisOrchestrator {
	go_ := &GenesisOrchestrator{
		liquidBrain: NewLiquidStateBrain(size),
		neurons:     make(map[string]*OrchestratorNeuron),
		decisions:   make(chan Decision, 100),
	}
	
	// Register capabilities as special neurons
	go_.RegisterCapability("gpt4", mockGPT4)
	go_.RegisterCapability("claude", mockClaude)
	go_.RegisterCapability("calculator", mockCalculator)
	go_.RegisterCapability("database", mockDatabase)
	
	return go_
}

func (go_ *GenesisOrchestrator) RegisterCapability(name string, endpoint func(context.Context, string) (string, error)) {
	go_.mu.Lock()
	defer go_.mu.Unlock()
	
	// Create special neurons for each capability
	neuron := &OrchestratorNeuron{
		capability: name,
		endpoint:   endpoint,
	}
	go_.neurons[name] = neuron
}

func (go_ *GenesisOrchestrator) Process(input string) (string, []Decision) {
	ctx := context.Background()
	decisions := []Decision{}
	
	// Phase 1: Liquid brain understands the input
	fmt.Printf("\n🧠 UNDERSTANDING: Processing through liquid neural reservoir...\n")
	understanding := go_.liquidBrain.Think(input)
	
	decision := Decision{
		Input:     input,
		Path:      []string{"liquid_brain"},
		Reasoning: "Initial understanding through parallel neural processing",
		Output:    understanding,
		Timestamp: time.Now(),
	}
	decisions = append(decisions, decision)
	
	// Phase 2: Route to appropriate capabilities based on understanding
	fmt.Printf("\n🔄 ROUTING: Determining which capabilities to engage...\n")
	
	// Simple routing logic (in production, this would be learned)
	var finalOutput string
	if containsAny(input, []string{"calculate", "math", "number"}) {
		fmt.Printf("   → Routing to calculator\n")
		result, _ := go_.neurons["calculator"].endpoint(ctx, input)
		finalOutput = result
		decisions = append(decisions, Decision{
			Input:     input,
			Path:      []string{"liquid_brain", "calculator"},
			Reasoning: "Detected mathematical intent",
			Output:    result,
			Timestamp: time.Now(),
		})
	} else if containsAny(input, []string{"creative", "story", "write"}) {
		fmt.Printf("   → Routing to Claude for creativity\n")
		result, _ := go_.neurons["claude"].endpoint(ctx, input)
		finalOutput = result
		decisions = append(decisions, Decision{
			Input:     input,
			Path:      []string{"liquid_brain", "claude"},
			Reasoning: "Detected creative intent",
			Output:    result,
			Timestamp: time.Now(),
		})
	} else if containsAny(input, []string{"data", "query", "find"}) {
		fmt.Printf("   → Routing to database\n")
		result, _ := go_.neurons["database"].endpoint(ctx, input)
		finalOutput = result
		decisions = append(decisions, Decision{
			Input:     input,
			Path:      []string{"liquid_brain", "database"},
			Reasoning: "Detected data query intent",
			Output:    result,
			Timestamp: time.Now(),
		})
	} else {
		fmt.Printf("   → Routing to GPT-4 for general query\n")
		result, _ := go_.neurons["gpt4"].endpoint(ctx, input)
		finalOutput = result
		decisions = append(decisions, Decision{
			Input:     input,
			Path:      []string{"liquid_brain", "gpt4"},
			Reasoning: "General query - using GPT-4",
			Output:    result,
			Timestamp: time.Now(),
		})
	}
	
	// Phase 3: Show complete decision trace
	fmt.Printf("\n📊 DECISION TRACE:\n")
	for i, d := range decisions {
		fmt.Printf("   Step %d: %s → %s\n", i+1, d.Path[len(d.Path)-1], d.Reasoning)
	}
	
	return finalOutput, decisions
}

func containsAny(s string, words []string) bool {
	for _, word := range words {
		if len(s) >= len(word) {
			for i := 0; i <= len(s)-len(word); i++ {
				if s[i:i+len(word)] == word {
					return true
				}
			}
		}
	}
	return false
}

// DemoOrchestration shows how Genesis orchestrates different AI systems
func DemoOrchestration() {
	fmt.Println("\n🎭 Genesis Orchestration Demo")
	fmt.Println("=" + strings.Repeat("=", 49))
	
	orchestrator := NewGenesisOrchestrator(1000)
	defer orchestrator.liquidBrain.Cleanup()
	
	// Test different types of requests
	tests := []string{
		"calculate the square root of 144",
		"write a creative story about robots",
		"find user data for John Doe",
		"explain quantum computing",
	}
	
	for _, test := range tests {
		fmt.Printf("\n\n💬 USER: %s\n", test)
		output, decisions := orchestrator.Process(test)
		fmt.Printf("\n✅ FINAL OUTPUT: %s\n", output)
		
		// In production, these decisions would be logged for monitoring
		orchestrator.logDecisions(decisions)
	}
}

func (go_ *GenesisOrchestrator) logDecisions(decisions []Decision) {
	// In production: Send to monitoring system, store in database, etc.
	// For demo: Just count them
	go_.mu.Lock()
	for _, d := range decisions {
		go_.decisions <- d
	}
	go_.mu.Unlock()
}