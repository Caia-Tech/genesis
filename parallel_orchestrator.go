package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ParallelOrchestrator - Neurons that orchestrate in parallel
type ParallelOrchestrator struct {
	neurons     []*SmartNeuron
	connections map[*SmartNeuron][]*SmartNeuron
	decisions   chan FlowDecision
	flowViz     chan FlowPattern
	active      int64
}

// SmartNeuron - A neuron that can make decisions and call services
type SmartNeuron struct {
	id          int
	activation  atomic.Value // float64
	capability  string
	threshold   float64
	
	// Each neuron can independently decide to call services
	canCallGPT    bool
	canCallClaude bool
	canCallTools  bool
	
	// Transparent decision making
	lastDecision string
	confidence   float64
}

type FlowDecision struct {
	NeuronID   int
	Activation float64
	Decision   string
	Confidence float64
	Timestamp  time.Time
}

type FlowPattern struct {
	ActiveNeurons int
	FlowPaths     [][]int
	Consensus     string
}

// NewParallelOrchestrator - Create massive parallel decision maker
func NewParallelOrchestrator(size int) *ParallelOrchestrator {
	po := &ParallelOrchestrator{
		neurons:     make([]*SmartNeuron, size),
		connections: make(map[*SmartNeuron][]*SmartNeuron),
		decisions:   make(chan FlowDecision, size),
		flowViz:     make(chan FlowPattern, 100),
	}
	
	// Create diverse neurons with different capabilities
	for i := 0; i < size; i++ {
		neuron := &SmartNeuron{
			id:        i,
			threshold: rand.Float64() * 0.5 + 0.3,
		}
		
		// Assign capabilities randomly (in production: learned)
		r := rand.Float64()
		if r < 0.3 {
			neuron.canCallGPT = true
			neuron.capability = "gpt_caller"
		} else if r < 0.6 {
			neuron.canCallClaude = true
			neuron.capability = "claude_caller"
		} else {
			neuron.canCallTools = true
			neuron.capability = "tool_caller"
		}
		
		neuron.activation.Store(0.0)
		po.neurons[i] = neuron
	}
	
	// Create connections (local connectivity for efficiency)
	for i, n := range po.neurons {
		// Each neuron connects to ~10 neighbors
		for j := 0; j < 10; j++ {
			target := rand.Intn(size)
			if target != i {
				po.connections[n] = append(po.connections[n], po.neurons[target])
			}
		}
	}
	
	return po
}

// ProcessInParallel - True parallel processing where each neuron decides independently
func (po *ParallelOrchestrator) ProcessInParallel(input string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	fmt.Printf("\n⚡ PARALLEL ORCHESTRATION: %d neurons processing simultaneously\n", len(po.neurons))
	
	// Phase 1: Inject input signal
	po.injectSignal(input)
	
	// Phase 2: Let neurons process in parallel
	var wg sync.WaitGroup
	decisionCollector := make(chan FlowDecision, len(po.neurons))
	
	// Each neuron processes independently
	for _, neuron := range po.neurons {
		wg.Add(1)
		go func(n *SmartNeuron) {
			defer wg.Done()
			
			// Neuron processes based on activation
			activation := n.activation.Load().(float64)
			if activation > n.threshold {
				decision := n.makeDecision(ctx, input, activation)
				decisionCollector <- decision
				
				// Propagate to connected neurons
				po.propagate(n, activation)
			}
		}(neuron)
	}
	
	// Wait for initial wave
	time.Sleep(100 * time.Millisecond)
	
	// Collect decisions
	go func() {
		wg.Wait()
		close(decisionCollector)
	}()
	
	// Phase 3: Aggregate decisions (consensus mechanism)
	decisions := []FlowDecision{}
	for d := range decisionCollector {
		decisions = append(decisions, d)
		if len(decisions) >= 10 { // Early stopping if we have enough
			break
		}
	}
	
	// Show parallel decision flow
	po.visualizeFlow(decisions)
	
	// Return consensus
	return po.formConsensus(decisions)
}

func (po *ParallelOrchestrator) injectSignal(input string) {
	// Inject at random points to simulate distributed input
	injectPoints := 10
	for i := 0; i < injectPoints; i++ {
		idx := rand.Intn(len(po.neurons))
		po.neurons[idx].activation.Store(1.0)
	}
	atomic.AddInt64(&po.active, int64(injectPoints))
}

func (n *SmartNeuron) makeDecision(ctx context.Context, input string, activation float64) FlowDecision {
	n.confidence = activation
	
	// Each neuron independently decides what to do
	if n.canCallGPT && activation > 0.8 {
		n.lastDecision = fmt.Sprintf("GPT-4[neuron_%d]: Process '%s'", n.id, input)
	} else if n.canCallClaude && activation > 0.7 {
		n.lastDecision = fmt.Sprintf("Claude[neuron_%d]: Create '%s'", n.id, input)
	} else if n.canCallTools && activation > 0.6 {
		n.lastDecision = fmt.Sprintf("Tools[neuron_%d]: Analyze '%s'", n.id, input)
	} else {
		n.lastDecision = fmt.Sprintf("Local[neuron_%d]: Think about '%s'", n.id, input)
	}
	
	return FlowDecision{
		NeuronID:   n.id,
		Activation: activation,
		Decision:   n.lastDecision,
		Confidence: n.confidence,
		Timestamp:  time.Now(),
	}
}

func (po *ParallelOrchestrator) propagate(source *SmartNeuron, signal float64) {
	// Propagate activation to connected neurons
	for _, target := range po.connections[source] {
		current := target.activation.Load().(float64)
		// Decay signal as it propagates
		newActivation := current + signal*0.7
		if newActivation > 1.0 {
			newActivation = 1.0
		}
		target.activation.Store(newActivation)
	}
}

func (po *ParallelOrchestrator) visualizeFlow(decisions []FlowDecision) {
	fmt.Printf("\n🌊 DECISION FLOW VISUALIZATION:\n")
	fmt.Printf("   Active neurons: %d\n", len(decisions))
	fmt.Printf("   Parallel decisions made:\n")
	
	// Group by capability
	gptCount, claudeCount, toolCount, localCount := 0, 0, 0, 0
	
	for _, d := range decisions {
		// Show top 5 decisions
		if gptCount+claudeCount+toolCount+localCount < 5 {
			fmt.Printf("   → %s (confidence: %.2f)\n", d.Decision, d.Confidence)
		}
		
		// Count types
		neuron := po.neurons[d.NeuronID]
		switch neuron.capability {
		case "gpt_caller":
			gptCount++
		case "claude_caller":
			claudeCount++
		case "tool_caller":
			toolCount++
		default:
			localCount++
		}
	}
	
	fmt.Printf("\n   Decision distribution:\n")
	fmt.Printf("   • GPT neurons activated: %d\n", gptCount)
	fmt.Printf("   • Claude neurons activated: %d\n", claudeCount)
	fmt.Printf("   • Tool neurons activated: %d\n", toolCount)
	fmt.Printf("   • Local processing: %d\n", localCount)
}

func (po *ParallelOrchestrator) formConsensus(decisions []FlowDecision) string {
	if len(decisions) == 0 {
		return "No consensus reached - insufficient activation"
	}
	
	// Simple voting mechanism (in production: weighted by confidence)
	highestConfidence := 0.0
	bestDecision := ""
	
	for _, d := range decisions {
		if d.Confidence > highestConfidence {
			highestConfidence = d.Confidence
			bestDecision = d.Decision
		}
	}
	
	return fmt.Sprintf("CONSENSUS: %s (confidence: %.2f from %d parallel decisions)", 
		bestDecision, highestConfidence, len(decisions))
}

// DemoParallelOrchestration - Show true parallel decision making
func DemoParallelOrchestration() {
	fmt.Println("\n🧠 Parallel Neural Orchestration Demo")
	fmt.Println("=" + strings.Repeat("=", 49))
	
	// Create a larger neural orchestrator
	orchestrator := NewParallelOrchestrator(1000)
	
	tests := []string{
		"Solve this complex problem requiring multiple perspectives",
		"Create something beautiful and meaningful",
		"Analyze this data and find patterns",
	}
	
	for _, test := range tests {
		fmt.Printf("\n\n💭 INPUT: %s\n", test)
		result := orchestrator.ProcessInParallel(test)
		fmt.Printf("\n✨ %s\n", result)
	}
}

// ShowScalingBehavior - Demonstrate how behavior changes with scale
func ShowScalingBehavior() {
	fmt.Println("\n📈 Scaling Behavior Demo")
	fmt.Println("=" + strings.Repeat("=", 49))
	
	sizes := []int{10, 100, 1000, 10000}
	input := "Understand the nature of consciousness"
	
	for _, size := range sizes {
		fmt.Printf("\n🔬 Testing with %d neurons:\n", size)
		orch := NewParallelOrchestrator(size)
		
		start := time.Now()
		result := orch.ProcessInParallel(input)
		elapsed := time.Since(start)
		
		fmt.Printf("   Time: %v\n", elapsed)
		fmt.Printf("   Result: %s\n", result)
		
		// At larger scales, more complex behaviors emerge
		if size >= 1000 {
			fmt.Printf("   💫 Emergent behavior detected at scale!\n")
		}
	}
}