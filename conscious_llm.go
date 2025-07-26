package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// TransparentLLM - An LLM that shows HOW it understands
type TransparentLLM struct {
	concepts      map[string]*ConceptNeuron
	activeCircuits map[string]*CircuitPath
	thoughtStream  chan ThoughtTrace
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	dataLoader    *DatasetLoader
	generator     *ResponseGenerator
}

type ConceptNeuron struct {
	id          string
	activation  atomic.Value // float64
	connections map[string]*Connection
	meaning     []float64 // semantic embedding
	visual      chan Pulse // for visualization
	ctx         context.Context
}

type Connection struct {
	to       *ConceptNeuron
	strength float64
	active   atomic.Value // bool
}

type Pulse struct {
	intensity float64
	source    string
	path      []string
}

type CircuitPath struct {
	nodes     []*ConceptNeuron
	strength  float64
	meaning   string
	timestamp time.Time
}

type ThoughtTrace struct {
	stage    string
	circuits []CircuitPath
	insight  string
}

func NewTransparentLLM() *TransparentLLM {
	config := DefaultConfig()
	return NewTransparentLLMWithConfig(config)
}

func NewTransparentLLMWithConfig(config *Config) *TransparentLLM {
	if config == nil {
		config = DefaultConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	llm := &TransparentLLM{
		concepts:       make(map[string]*ConceptNeuron),
		activeCircuits: make(map[string]*CircuitPath),
		thoughtStream:  make(chan ThoughtTrace, config.Resources.ChannelBufferSize),
		ctx:            ctx,
		cancel:         cancel,
	}
	
	// Load dataset with error handling
	dataLoader, err := NewDatasetLoader(config.Training)
	if err != nil {
		fmt.Printf("⚠️  Warning: failed to load dataset: %v\n", err)
		fmt.Println("🔄 Falling back to basic concepts...")
		if initErr := llm.initializeConceptNetwork(); initErr != nil {
			fmt.Printf("❌ ERROR: Failed to initialize concept network: %v\n", initErr)
			return nil
		}
	} else {
		llm.dataLoader = dataLoader
		llm.generator = NewResponseGenerator(dataLoader)
		llm.initializeFromDataset(config)
	}
	
	return llm
}

// Cleanup properly shuts down the LLM with timeout
func (llm *TransparentLLM) Cleanup() {
	if llm.cancel == nil {
		return // Already cleaned up
	}
	
	fmt.Println("🔄 Initiating LLM cleanup...")
	llm.cancel()
	
	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		llm.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		fmt.Println("✅ All LLM goroutines terminated gracefully")
	case <-time.After(3 * time.Second):
		fmt.Println("⚠️  LLM cleanup timeout - some goroutines may still be running")
	}
	
	// Safely close channels and clear resources
	if llm.thoughtStream != nil {
		close(llm.thoughtStream)
		llm.thoughtStream = nil
	}
	
	// Clear concept neurons' visual channels
	for _, neuron := range llm.concepts {
		if neuron.visual != nil {
			close(neuron.visual)
			neuron.visual = nil
		}
	}
	
	llm.cancel = nil // Mark as cleaned up
	fmt.Println("✅ LLM cleanup completed")
}

func (llm *TransparentLLM) initializeConceptNetwork() error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("🚨 Concept network initialization panic recovered: %v\n", r)
		}
	}()
	// Create a rich semantic network
	concepts := []string{
		"question", "understand", "meaning", "context",
		"user", "intent", "emotion", "help", "solve",
		"pattern", "similar", "experience", "connection",
		"frustration", "code", "debug", "error", "stuck",
		"insight", "solution", "approach", "alternative",
	}
	
	// Create neurons for each concept
	for _, concept := range concepts {
		neuron := &ConceptNeuron{
			id:          concept,
			connections: make(map[string]*Connection),
			meaning:     generateSemanticVector(concept),
			visual:      make(chan Pulse, 10), // Reduced buffer
			ctx:         llm.ctx,
		}
		neuron.activation.Store(0.0)
		llm.concepts[concept] = neuron
		
		// Start neuron's autonomous processing with error handling
		llm.wg.Add(1)
		go func(n *ConceptNeuron) {
			defer llm.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("🚨 Concept neuron panic recovered: %v\n", r)
				}
			}()
			n.live()
		}(neuron)
	}
	
	// Create meaningful connections
	llm.connect("question", "understand", 0.9)
	llm.connect("question", "intent", 0.8)
	llm.connect("understand", "meaning", 0.9)
	llm.connect("understand", "context", 0.7)
	llm.connect("user", "intent", 0.8)
	llm.connect("user", "emotion", 0.6)
	llm.connect("frustration", "stuck", 0.9)
	llm.connect("frustration", "error", 0.8)
	llm.connect("code", "debug", 0.7)
	llm.connect("code", "error", 0.8)
	llm.connect("stuck", "help", 0.9)
	llm.connect("help", "solution", 0.8)
	llm.connect("pattern", "similar", 0.9)
	llm.connect("pattern", "experience", 0.7)
	llm.connect("insight", "solution", 0.8)
	llm.connect("insight", "approach", 0.7)
	
	fmt.Printf("✅ Initialized %d concept neurons\n", len(llm.concepts))
	return nil
}

func (llm *TransparentLLM) initializeFromDataset(config *Config) {
	vocab := llm.dataLoader.GetVocabulary()
	fmt.Printf("Initializing network with %d concepts from dataset\n", len(vocab))
	
	// Limit concepts to configured maximum
	maxConcepts := config.Model.MaxConcepts
	if len(vocab) > maxConcepts {
		vocab = vocab[:maxConcepts]
	}
	
	// Create neurons for vocabulary words
	for _, word := range vocab {
		embedding, _ := llm.dataLoader.GetEmbedding(word)
		neuron := &ConceptNeuron{
			id:          word,
			connections: make(map[string]*Connection),
			meaning:     embedding,
			visual:      make(chan Pulse, config.Resources.ChannelBufferSize/10),
			ctx:         llm.ctx,
		}
		neuron.activation.Store(0.0)
		llm.concepts[word] = neuron
		
		// Start neuron's autonomous processing
		llm.wg.Add(1)
		go func(n *ConceptNeuron) {
			defer llm.wg.Done()
			n.live()
		}(neuron)
	}
	
	// Create connections based on semantic similarity
	llm.createSemanticConnections()
}

func (llm *TransparentLLM) connect(from, to string, strength float64) {
	fromNeuron := llm.concepts[from]
	toNeuron := llm.concepts[to]
	
	if fromNeuron == nil || toNeuron == nil {
		return
	}
	
	fromNeuron.connections[to] = &Connection{
		to:       toNeuron,
		strength: strength,
	}
	
	// Bidirectional with slightly less strength
	toNeuron.connections[from] = &Connection{
		to:       fromNeuron,
		strength: strength * 0.7,
	}
}

func (llm *TransparentLLM) createSemanticConnections() {
	vocab := llm.dataLoader.GetVocabulary()
	connectionCount := 0
	maxConnectionsPerWord := 10
	
	// Create connections based on semantic similarity
	for i, word1 := range vocab {
		bestConnections := make([]struct {
			word string
			sim  float64
		}, 0, maxConnectionsPerWord)
		
		// Find most similar words
		for j, word2 := range vocab {
			if i == j {
				continue
			}
			
			sim := llm.dataLoader.ComputeSimilarity(word1, word2)
			if sim > 0.5 {
				bestConnections = append(bestConnections, struct {
					word string
					sim  float64
				}{word2, sim})
			}
		}
		
		// Sort by similarity and keep top connections
		if len(bestConnections) > maxConnectionsPerWord {
			// Simple selection of top connections
			bestConnections = bestConnections[:maxConnectionsPerWord]
		}
		
		// Create connections
		for _, conn := range bestConnections {
			llm.connect(word1, conn.word, conn.sim)
			connectionCount++
		}
	}
	
	fmt.Printf("Created %d semantic connections\n", connectionCount)
}

// The magic happens here - WATCH the understanding process
func (llm *TransparentLLM) Understand(input string) (string, <-chan ThoughtTrace) {
	fmt.Println("\n🧠 Watch as I understand your question...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	// Create visualization channel
	visualization := make(chan ThoughtTrace, 100)
	var processingDone sync.WaitGroup
	var response string
	
	processingDone.Add(1)
	go func() {
		defer processingDone.Done()
		
		// Stage 1: Parallel word activation
		llm.thoughtStream <- ThoughtTrace{
			stage:   "PARSING",
			insight: "Activating word concepts in parallel...",
		}
		
		words := strings.Fields(strings.ToLower(input))
		var wg sync.WaitGroup
		
		// Every word creates ripples through the network
		for _, word := range words {
			wg.Add(1)
			go func(w string) {
				defer wg.Done()
				llm.activateWord(w)
			}(word)
		}
		
		wg.Wait()
		time.Sleep(50 * time.Millisecond) // Let activation spread
		
		// Stage 2: Pattern emergence
		llm.thoughtStream <- ThoughtTrace{
			stage:   "PATTERN_RECOGNITION",
			insight: "Watching for emerging patterns...",
		}
		
		// Find active circuits
		circuits := llm.findActiveCircuits()
		
		llm.thoughtStream <- ThoughtTrace{
			stage:    "CIRCUITS_FOUND",
			circuits: circuits,
			insight:  fmt.Sprintf("Found %d active meaning circuits", len(circuits)),
		}
		
		// Stage 3: Meaning crystallization
		dominantMeaning := llm.crystallizeMeaning(circuits)
		
		llm.thoughtStream <- ThoughtTrace{
			stage:   "UNDERSTANDING",
			insight: fmt.Sprintf("Primary understanding: %s", dominantMeaning),
		}
		
		// Stage 4: Response generation with visible reasoning
		response = llm.generateResponse(dominantMeaning, circuits)
		
		llm.thoughtStream <- ThoughtTrace{
			stage:   "RESPONSE_GENERATION",
			insight: fmt.Sprintf("Generated response: %s", response),
		}
	}()
	
	// Stream thoughts to visualization
	processingDone.Add(1)
	go func() {
		defer processingDone.Done()
		defer close(visualization)
		
		for {
			select {
			case thought, ok := <-llm.thoughtStream:
				if !ok {
					return
				}
				visualization <- thought
				llm.visualizeThought(thought)
			case <-time.After(2 * time.Second):
				// Timeout in case processing gets stuck
				return
			}
		}
	}()
	
	// Wait for processing to complete
	processingDone.Wait()
	
	return response, visualization
}

func (llm *TransparentLLM) activateWord(word string) {
	// Direct activation
	if neuron, exists := llm.concepts[word]; exists {
		neuron.activate(1.0)
		
		// Send pulse for visualization
		pulse := Pulse{
			intensity: 1.0,
			source:    word,
			path:      []string{word},
		}
		
		select {
		case neuron.visual <- pulse:
		case <-time.After(1 * time.Millisecond):
			// Channel blocked, skip to prevent deadlock
		default:
			// Channel full, skip this pulse
		}
	}
	
	// Semantic activation - find related concepts
	for concept, neuron := range llm.concepts {
		similarity := llm.semanticSimilarity(word, concept)
		if similarity > 0.5 {
			neuron.activate(similarity)
		}
	}
}

func (llm *TransparentLLM) findActiveCircuits() []CircuitPath {
	circuits := []CircuitPath{}
	
	// Use parallel search for circuit detection
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	for _, startNeuron := range llm.concepts {
		if startNeuron.getActivation() > 0.5 {
			wg.Add(1)
			go func(start *ConceptNeuron) {
				defer wg.Done()
				
				// Trace active paths from this neuron
				paths := llm.tracePaths(start, []string{start.id}, 0.5)
				
				mu.Lock()
				circuits = append(circuits, paths...)
				mu.Unlock()
			}(startNeuron)
		}
	}
	
	wg.Wait()
	return circuits
}

func (llm *TransparentLLM) tracePaths(current *ConceptNeuron, path []string, minStrength float64) []CircuitPath {
	circuits := []CircuitPath{}
	
	// Stop if path is too long or we're in a loop
	if len(path) > 5 || contains(path[:len(path)-1], current.id) {
		return circuits
	}
	
	// Check each connection
	for _, conn := range current.connections {
		if conn.to.getActivation() > minStrength {
			newPath := append(path, conn.to.id)
			
			// This is a meaningful circuit
			circuit := CircuitPath{
				nodes:     llm.getNodes(newPath),
				strength:  llm.calculatePathStrength(newPath),
				meaning:   strings.Join(newPath, "→"),
				timestamp: time.Now(),
			}
			
			circuits = append(circuits, circuit)
			
			// Continue tracing
			deeperCircuits := llm.tracePaths(conn.to, newPath, minStrength*0.8)
			circuits = append(circuits, deeperCircuits...)
		}
	}
	
	return circuits
}

func (llm *TransparentLLM) crystallizeMeaning(circuits []CircuitPath) string {
	// Find strongest pattern
	strongestPattern := ""
	maxStrength := 0.0
	
	patternStrength := make(map[string]float64)
	var mu sync.Mutex
	
	// Aggregate circuit patterns
	for _, circuit := range circuits {
		pattern := extractPattern(circuit)
		mu.Lock()
		patternStrength[pattern] += circuit.strength
		
		if patternStrength[pattern] > maxStrength {
			maxStrength = patternStrength[pattern]
			strongestPattern = pattern
		}
		mu.Unlock()
	}
	
	return strongestPattern
}

func (llm *TransparentLLM) generateResponse(meaning string, circuits []CircuitPath) string {
	// Use activated concepts to generate a response
	if llm.dataLoader == nil || llm.generator == nil {
		// Fallback to simple responses
		return llm.generateSimpleResponse(meaning, circuits)
	}
	
	// Get most activated concepts
	activeConcepts := llm.getTopActivatedConcepts(10)
	
	// Use the enhanced response generator
	response := llm.generator.Generate(meaning, activeConcepts)
	
	return response
}

func (llm *TransparentLLM) selectNextWord(currentWord string, activeConcepts []string, recent map[string]int) string {
	// Get transition candidates
	transitions, exists := llm.dataLoader.GetTransitions(currentWord)
	if !exists || len(transitions) == 0 {
		// Fallback: use an activated concept
		if len(activeConcepts) > 1 {
			return activeConcepts[rand.Intn(len(activeConcepts))]
		}
		return ""
	}
	
	// Score each candidate
	type candidate struct {
		word  string
		score float64
	}
	candidates := []candidate{}
	
	for nextWord, prob := range transitions {
		// Skip if used too recently
		if recent[nextWord] > 1 {
			continue
		}
		
		score := prob
		
		// Boost score if word is semantically related to active concepts
		for _, concept := range activeConcepts {
			if similarity := llm.semanticSimilarity(nextWord, concept); similarity > 0.3 {
				score *= (1.0 + similarity)
			}
		}
		
		// Slightly penalize very common words
		if nextWord == "the" || nextWord == "a" || nextWord == "is" {
			score *= 0.8
		}
		
		candidates = append(candidates, candidate{nextWord, score})
	}
	
	if len(candidates) == 0 {
		return ""
	}
	
	// Select based on scores (simple greedy for now)
	bestCandidate := candidates[0]
	for _, c := range candidates {
		if c.score > bestCandidate.score {
			bestCandidate = c
		}
	}
	
	return bestCandidate.word
}

func (llm *TransparentLLM) getTopActivatedConcepts(n int) []string {
	type conceptActivation struct {
		concept    string
		activation float64
	}
	
	activations := []conceptActivation{}
	
	for concept, neuron := range llm.concepts {
		if act := neuron.getActivation(); act > 0.1 {
			activations = append(activations, conceptActivation{concept, act})
		}
	}
	
	// Sort by activation
	for i := 0; i < len(activations); i++ {
		for j := i + 1; j < len(activations); j++ {
			if activations[j].activation > activations[i].activation {
				activations[i], activations[j] = activations[j], activations[i]
			}
		}
	}
	
	// Get top N
	result := []string{}
	for i := 0; i < n && i < len(activations); i++ {
		result = append(result, activations[i].concept)
	}
	
	return result
}

func (llm *TransparentLLM) generateSimpleResponse(meaning string, circuits []CircuitPath) string {
	// Fallback for when no dataset is loaded
	switch {
	case strings.Contains(meaning, "frustration") && strings.Contains(meaning, "code"):
		return "I understand code frustration. Let me help debug the issue."
	case strings.Contains(meaning, "help") && strings.Contains(meaning, "solution"):
		return "I'll help find a solution. What specific challenge are you facing?"
	default:
		return "I'm processing your input. Tell me more."
	}
}


func (llm *TransparentLLM) visualizeThought(thought ThoughtTrace) {
	switch thought.stage {
	case "PARSING":
		fmt.Println("\n⚡ PARSING:", thought.insight)
		
	case "PATTERN_RECOGNITION":
		fmt.Println("\n🔄 PATTERN RECOGNITION:", thought.insight)
		
	case "CIRCUITS_FOUND":
		fmt.Println("\n🧩 ACTIVE CIRCUITS:")
		for _, circuit := range thought.circuits[:min(5, len(thought.circuits))] {
			fmt.Printf("   → %s (strength: %.2f)\n", circuit.meaning, circuit.strength)
		}
		
	case "UNDERSTANDING":
		fmt.Println("\n💡 UNDERSTANDING:", thought.insight)
	case "RESPONSE_GENERATION":
		fmt.Println("\n💬 RESPONSE:", thought.insight)
	}
}

// Neuron methods
func (n *ConceptNeuron) live() {
	decay := 0.95
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-n.ctx.Done():
			return
		case pulse := <-n.visual:
			// Propagate activation
			n.activate(pulse.intensity)
			
			// Spread to connections
			for _, conn := range n.connections {
				if rand.Float64() < conn.strength {
					newPulse := Pulse{
						intensity: pulse.intensity * conn.strength,
						source:    pulse.source,
						path:      append(pulse.path, conn.to.id),
					}
					
					select {
					case conn.to.visual <- newPulse:
					default:
					}
				}
			}
			
		case <-ticker.C:
			// Decay activation
			current := n.getActivation()
			n.activation.Store(current * decay)
		}
	}
}

func (n *ConceptNeuron) activate(amount float64) {
	current := n.getActivation()
	n.activation.Store(math.Min(1.0, current+amount))
}

func (n *ConceptNeuron) getActivation() float64 {
	if val := n.activation.Load(); val != nil {
		return val.(float64)
	}
	return 0.0
}

// Helper functions
func generateSemanticVector(word string) []float64 {
	// Simplified semantic embedding
	vec := make([]float64, 64)
	for i := range vec {
		vec[i] = rand.Float64()
	}
	return vec
}

var (
	similaritiesOnce sync.Once
	similarities    map[string]map[string]float64
	similaritiesMu  sync.RWMutex
)

func initSimilarities() {
	similaritiesOnce.Do(func() {
		similarities = map[string]map[string]float64{
			"frustration": {"stuck": 0.8, "error": 0.7, "problem": 0.8},
			"code":        {"programming": 0.9, "debug": 0.7, "error": 0.6},
			"help":        {"assist": 0.9, "support": 0.8, "solve": 0.7},
		}
	})
}

func (llm *TransparentLLM) semanticSimilarity(word1, word2 string) float64 {
	// Use dataset embeddings if available
	if llm.dataLoader != nil {
		return llm.dataLoader.ComputeSimilarity(word1, word2)
	}
	
	// Fallback to hardcoded similarities
	initSimilarities()
	
	// Simplified similarity calculation
	if strings.Contains(word1, word2) || strings.Contains(word2, word1) {
		return 0.8
	}
	
	similaritiesMu.RLock()
	defer similaritiesMu.RUnlock()
	
	if sim, ok := similarities[word1][word2]; ok {
		return sim
	}
	if sim, ok := similarities[word2][word1]; ok {
		return sim
	}
	
	return 0.0
}

func semanticSimilarity(word1, word2 string) float64 {
	// Legacy function for compatibility
	llm := &TransparentLLM{}
	return llm.semanticSimilarity(word1, word2)
}

func (llm *TransparentLLM) getNodes(path []string) []*ConceptNeuron {
	nodes := []*ConceptNeuron{}
	for _, id := range path {
		if neuron, ok := llm.concepts[id]; ok {
			nodes = append(nodes, neuron)
		}
	}
	return nodes
}

func (llm *TransparentLLM) calculatePathStrength(path []string) float64 {
	if len(path) < 2 {
		return 0.0
	}
	
	strength := 1.0
	for i := 0; i < len(path)-1; i++ {
		from := llm.concepts[path[i]]
		if conn, ok := from.connections[path[i+1]]; ok {
			strength *= conn.strength
		} else {
			strength *= 0.1
		}
	}
	
	return strength
}

func extractPattern(circuit CircuitPath) string {
	// Extract the key pattern from a circuit
	if len(circuit.nodes) >= 2 {
		return circuit.nodes[0].id + "_" + circuit.nodes[len(circuit.nodes)-1].id
	}
	return ""
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// min function moved to utils.go to avoid duplicates