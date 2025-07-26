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

// LiquidStateBrain - A massive parallel brain that thinks like water
type LiquidStateBrain struct {
	reservoir    [][][]*LiquidNeuron // 3D neural reservoir
	dimensions   Dimensions
	inputLayer   []*InputNeuron
	outputLayer  []*OutputNeuron
	wavePatterns chan WavePattern
	thoughts     chan string
	activeWaves  int64
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	dataLoader   *DatasetLoader
	config       *Config
	generator    *ResponseGenerator
}

type Dimensions struct {
	X, Y, Z int
}

type LiquidNeuron struct {
	x, y, z      int
	state        atomic.Value // float64
	threshold    float64
	connections  []*LiquidNeuron
	lastFired    time.Time
	refractoryMs int64
	ctx          context.Context
}

type InputNeuron struct {
	connections []*LiquidNeuron
	word        string
}

type OutputNeuron struct {
	connections []*LiquidNeuron
	meaning     string
	activation  atomic.Value // float64
}

type WavePattern struct {
	origin    [3]int
	intensity float64
	timestamp time.Time
	meaning   string
}

func NewLiquidStateBrain(size int) *LiquidStateBrain {
	config := DefaultConfig()
	return NewLiquidStateBrainWithConfig(size, config)
}

func NewLiquidStateBrainWithConfig(size int, config *Config) *LiquidStateBrain {
	// Validate input parameters
	if size <= 0 {
		return nil
	}
	if config == nil {
		config = DefaultConfig()
	}
	
	// Apply strict resource limits to prevent OOM
	totalNeurons := size * size * (size / 2)
	if totalNeurons > config.Resources.MaxNeurons {
		// Calculate safe dimensions
		maxDim := int(math.Pow(float64(config.Resources.MaxNeurons*2), 1.0/3.0))
		if maxDim < 2 {
			maxDim = 2 // Minimum viable brain size
		}
		size = maxDim
		totalNeurons = size * size * (size / 2)
		fmt.Printf("⚠️  Adjusted brain size from original to %d neurons to prevent OOM\n", totalNeurons)
	}
	
	// Validate dimensions are reasonable
	if size < 2 {
		fmt.Printf("❌ ERROR: Brain size too small (%d), minimum is 2x2x1\n", size)
		return nil
	}
	
	// Additional safety check for memory usage
	estimatedMemoryMB := (totalNeurons * 200) / (1024 * 1024) // Rough estimate: 200 bytes per neuron
	if estimatedMemoryMB > config.Resources.MaxMemoryMB {
		fmt.Printf("❌ ERROR: Estimated memory usage (%d MB) exceeds limit (%d MB)\n", estimatedMemoryMB, config.Resources.MaxMemoryMB)
		return nil
	}
	
	dims := Dimensions{X: size, Y: size, Z: max(1, size/2)} // Ensure Z is at least 1
	ctx, cancel := context.WithCancel(context.Background())
	
	brain := &LiquidStateBrain{
		reservoir:    make([][][]*LiquidNeuron, dims.X),
		dimensions:   dims,
		wavePatterns: make(chan WavePattern, config.Resources.ChannelBufferSize),
		thoughts:     make(chan string, config.Resources.ChannelBufferSize/10),
		ctx:          ctx,
		cancel:       cancel,
		config:       config,
	}
	
	// Load dataset
	dataLoader, err := NewDatasetLoader(config.Training)
	if err != nil {
		fmt.Printf("Warning: failed to load dataset: %v\n", err)
	} else {
		brain.dataLoader = dataLoader
		brain.generator = NewResponseGenerator(dataLoader)
	}
	
	// Initialize 3D reservoir with progress tracking
	fmt.Printf("🌊 Initializing Liquid State Brain: %d x %d x %d = %d neurons\n",
		dims.X, dims.Y, dims.Z, dims.X*dims.Y*dims.Z)
	
	// Initialize with proper error handling
	neuronsCreated := 0
	for x := 0; x < dims.X; x++ {
		brain.reservoir[x] = make([][]*LiquidNeuron, dims.Y)
		for y := 0; y < dims.Y; y++ {
			brain.reservoir[x][y] = make([]*LiquidNeuron, dims.Z)
			for z := 0; z < dims.Z; z++ {
				// Check for context cancellation during initialization
				select {
				case <-brain.ctx.Done():
					fmt.Printf("❌ Brain initialization cancelled\n")
					return nil
				default:
				}
				
				neuron := &LiquidNeuron{
					x: x, y: y, z: z,
					threshold:    0.5 + rand.Float64()*0.3,
					refractoryMs: 5 + rand.Int63n(10),
					ctx:          brain.ctx,
					connections:  make([]*LiquidNeuron, 0, 10), // Pre-allocate with reasonable capacity
				}
				neuron.state.Store(rand.Float64() * 0.1)
				brain.reservoir[x][y][z] = neuron
				neuronsCreated++
				
				// Progress indicator for large brains
				if neuronsCreated%1000 == 0 {
					fmt.Printf("⚡ Created %d/%d neurons\n", neuronsCreated, dims.X*dims.Y*dims.Z)
				}
			}
		}
	}
	
	// Create local connections (nearby neurons)
	brain.connectReservoir()
	
	// Initialize input/output layers
	brain.initializeIO()
	
	// Start the liquid dynamics
	brain.startDynamics()
	
	return brain
}

func (brain *LiquidStateBrain) connectReservoir() {
	// Each neuron connects to nearby neurons
	radius := 2 // Connection radius
	
	for x := 0; x < brain.dimensions.X; x++ {
		for y := 0; y < brain.dimensions.Y; y++ {
			for z := 0; z < brain.dimensions.Z; z++ {
				neuron := brain.reservoir[x][y][z]
				
				// Connect to neighbors within radius
				for dx := -radius; dx <= radius; dx++ {
					for dy := -radius; dy <= radius; dy++ {
						for dz := -radius; dz <= radius; dz++ {
							if dx == 0 && dy == 0 && dz == 0 {
								continue
							}
							
							nx, ny, nz := x+dx, y+dy, z+dz
							
							// Check bounds
							if nx >= 0 && nx < brain.dimensions.X &&
							   ny >= 0 && ny < brain.dimensions.Y &&
							   nz >= 0 && nz < brain.dimensions.Z {
								
								// Probability of connection decreases with distance
								distance := math.Sqrt(float64(dx*dx + dy*dy + dz*dz))
								if rand.Float64() < 0.3/distance {
									neighbor := brain.reservoir[nx][ny][nz]
									neuron.connections = append(neuron.connections, neighbor)
								}
							}
						}
					}
				}
			}
		}
	}
	
	fmt.Printf("✓ Connected reservoir with local topology\n")
}

func (brain *LiquidStateBrain) initializeIO() {
	// Create input neurons for different concepts
	concepts := []string{"hello", "help", "code", "error", "think", "understand"}
	brain.inputLayer = make([]*InputNeuron, len(concepts))
	
	// Safety check for empty reservoir
	if brain.dimensions.X == 0 || brain.dimensions.Y == 0 || brain.dimensions.Z == 0 {
		fmt.Println("❌ ERROR: Cannot create I/O layers with zero-sized reservoir")
		return
	}
	
	for i, concept := range concepts {
		input := &InputNeuron{word: concept}
		
		// Connect to random neurons in first layer
		for j := 0; j < 100; j++ {
			x := rand.Intn(brain.dimensions.X)
			y := rand.Intn(brain.dimensions.Y)
			z := 0 // First layer
			input.connections = append(input.connections, brain.reservoir[x][y][z])
		}
		
		brain.inputLayer[i] = input
	}
	
	// Create output neurons for interpretations
	outputs := []string{"greeting", "assistance", "technical", "problem", "cognitive", "comprehension"}
	brain.outputLayer = make([]*OutputNeuron, len(outputs))
	
	for i, meaning := range outputs {
		output := &OutputNeuron{meaning: meaning}
		output.activation.Store(0.0)
		
		// Connect to random neurons in last layer
		for j := 0; j < 100; j++ {
			x := rand.Intn(brain.dimensions.X)
			y := rand.Intn(brain.dimensions.Y)
			z := brain.dimensions.Z - 1 // Last layer
			output.connections = append(output.connections, brain.reservoir[x][y][z])
		}
		
		brain.outputLayer[i] = output
	}
}

func (brain *LiquidStateBrain) startDynamics() {
	// Start reservoir dynamics with proper resource management
	totalNeurons := 0
	goroutineCount := 0
	maxGoroutines := brain.config.Resources.MaxGoroutines
	
	// Calculate neurons per goroutine to stay within limits
	neuronsPerGoroutine := 1
	totalNeuronsCount := brain.dimensions.X * brain.dimensions.Y * brain.dimensions.Z
	if totalNeuronsCount > maxGoroutines {
		neuronsPerGoroutine = (totalNeuronsCount + maxGoroutines - 1) / maxGoroutines
		fmt.Printf("⚡ Batching %d neurons per goroutine to stay within %d goroutine limit\n", neuronsPerGoroutine, maxGoroutines)
	}
	
	for x := 0; x < brain.dimensions.X; x++ {
		for y := 0; y < brain.dimensions.Y; y++ {
			for z := 0; z < brain.dimensions.Z; z++ {
				// Check if we need to start a new batch
				if totalNeurons%neuronsPerGoroutine == 0 && goroutineCount < maxGoroutines {
					// Start goroutine for batch of neurons
					batchStart := totalNeurons
					batchEnd := min(totalNeurons+neuronsPerGoroutine, totalNeuronsCount)
					
					brain.wg.Add(1)
					go func(start, end int) {
						defer brain.wg.Done()
						defer func() {
							if r := recover(); r != nil {
								fmt.Printf("🚨 Neuron goroutine panic recovered: %v\n", r)
							}
						}()
						
						// Run neurons in this batch
						neuronIndex := 0
						for bx := 0; bx < brain.dimensions.X && neuronIndex < end-start; bx++ {
							for by := 0; by < brain.dimensions.Y && neuronIndex < end-start; by++ {
								for bz := 0; bz < brain.dimensions.Z && neuronIndex < end-start; bz++ {
									if neuronIndex >= start-start { // Within our batch
										brain.reservoir[bx][by][bz].live()
									}
									neuronIndex++
								}
							}
						}
					}(batchStart, batchEnd)
					goroutineCount++
				}
				totalNeurons++
			}
		}
	}
	
	// Start output monitoring with error handling
	for _, output := range brain.outputLayer {
		brain.wg.Add(1)
		go func(o *OutputNeuron) {
			defer brain.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("🚨 Output monitor panic recovered: %v\n", r)
				}
			}()
			o.monitor(brain.ctx)
		}(output)
	}
	
	// Start wave visualization with error handling
	brain.wg.Add(1)
	go func() {
		defer brain.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("🚨 Wave visualization panic recovered: %v\n", r)
			}
		}()
		brain.visualizeWaves()
	}()
	
	fmt.Printf("🚀 Started %d neurons with %d goroutines\n", totalNeurons, goroutineCount)
}

// Cleanup properly shuts down the brain with timeout
func (brain *LiquidStateBrain) Cleanup() {
	if brain.cancel == nil {
		return // Already cleaned up
	}
	
	fmt.Println("🔄 Initiating brain cleanup...")
	brain.cancel()
	
	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		brain.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		fmt.Println("✅ All goroutines terminated gracefully")
	case <-time.After(5 * time.Second):
		fmt.Println("⚠️  Cleanup timeout - some goroutines may still be running")
	}
	
	// Safely close channels
	if brain.wavePatterns != nil {
		close(brain.wavePatterns)
		brain.wavePatterns = nil
	}
	if brain.thoughts != nil {
		close(brain.thoughts)
		brain.thoughts = nil
	}
	
	brain.cancel = nil // Mark as cleaned up
	fmt.Println("✅ Brain cleanup completed")
}

// Process input and watch patterns emerge
func (brain *LiquidStateBrain) Think(input string) string {
	fmt.Printf("\n🧠 Liquid brain processing: '%s'\n", input)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	// Inject input as waves
	words := strings.Fields(strings.ToLower(input))
	
	for _, word := range words {
		brain.injectWord(word)
	}
	
	// Let waves propagate
	time.Sleep(200 * time.Millisecond)
	
	// Generate response based on wave patterns
	response := brain.generateResponse()
	
	// Show active wave count
	waves := atomic.LoadInt64(&brain.activeWaves)
	fmt.Printf("\n📊 Active waves in reservoir: %d\n", waves)
	
	return response
}

func (brain *LiquidStateBrain) injectWord(word string) {
	// Find matching input neuron
	for _, input := range brain.inputLayer {
		similarity := brain.wordSimilarity(word, input.word)
		if similarity > 0.5 {
			// Create ripples from this input
			fmt.Printf("💉 Injecting '%s' (similarity to '%s': %.2f)\n", 
				word, input.word, similarity)
			
			// Stimulate connected neurons
			for _, neuron := range input.connections {
				go func(n *LiquidNeuron, strength float64) {
					defer func() {
						if r := recover(); r != nil {
							fmt.Printf("🚨 Neuron activation panic recovered: %v\n", r)
						}
					}()
					
					var current float64
					if val := n.state.Load(); val != nil {
						current = val.(float64)
					}
					n.state.Store(math.Min(1.0, current + strength))
					
					// Record wave pattern with non-blocking approach
					select {
					case brain.wavePatterns <- WavePattern{
						origin:    [3]int{n.x, n.y, n.z},
						intensity: strength,
						timestamp: time.Now(),
						meaning:   word,
					}:
						atomic.AddInt64(&brain.activeWaves, 1)
					case <-time.After(5 * time.Millisecond):
						// Channel blocked, skip this wave to prevent deadlock
					default:
						// Channel full, skip this wave
					}
				}(neuron, similarity)
			}
		}
	}
}

func (brain *LiquidStateBrain) readOutput() map[string]float64 {
	// Collect activation from output neurons
	activations := make(map[string]float64)
	
	for _, output := range brain.outputLayer {
		// Sum activation from connected neurons
		totalActivation := 0.0
		for _, neuron := range output.connections {
			if val := neuron.state.Load(); val != nil {
				totalActivation += val.(float64)
			}
		}
		
		avgActivation := totalActivation / float64(len(output.connections))
		activations[output.meaning] = avgActivation
	}
	
	// Show activation pattern
	fmt.Println("\n🎯 Output activations:")
	for meaning, activation := range activations {
		bar := strings.Repeat("█", int(activation*20))
		fmt.Printf("   %-15s [%-20s] %.2f\n", meaning, bar, activation)
	}
	
	return activations
}

func (brain *LiquidStateBrain) generateResponse() string {
	// Get output activations
	activations := brain.readOutput()
	
	if brain.dataLoader == nil || brain.generator == nil {
		// Fallback to simple interpretation
		return brain.simpleInterpretation(activations)
	}
	
	// Convert activations to concepts
	activeConcepts := brain.getActivatedConcepts(activations)
	
	// Build input context from wave patterns
	context := brain.getWaveContext()
	
	// Use enhanced generator
	response := brain.generator.Generate(context, activeConcepts)
	
	return response
}

func (brain *LiquidStateBrain) getActivatedConcepts(activations map[string]float64) []string {
	concepts := []string{}
	
	// Get strongly activated outputs
	for meaning, activation := range activations {
		if activation > 0.5 {
			// Map to related concepts
			switch meaning {
			case "greeting":
				concepts = append(concepts, "hello", "welcome", "greet")
			case "assistance":
				concepts = append(concepts, "help", "assist", "support", "guide")
			case "technical":
				concepts = append(concepts, "code", "system", "process", "compute")
			case "problem":
				concepts = append(concepts, "solve", "debug", "fix", "issue")
			case "cognitive":
				concepts = append(concepts, "think", "understand", "analyze", "reason")
			case "comprehension":
				concepts = append(concepts, "understand", "grasp", "see", "know")
			}
		}
	}
	
	return concepts
}

func (brain *LiquidStateBrain) getWaveContext() string {
	// Extract meaning from recent wave patterns
	recentWaves := []string{}
	
	// Sample recent wave patterns
	select {
	case wave := <-brain.wavePatterns:
		recentWaves = append(recentWaves, wave.meaning)
		// Put it back
		brain.wavePatterns <- wave
	default:
	}
	
	if len(recentWaves) > 0 {
		return strings.Join(recentWaves, " ")
	}
	
	return "waves flowing through reservoir"
}

// Legacy methods kept for compatibility
func (brain *LiquidStateBrain) getActivatedWords(activations map[string]float64) []string {
	words := []string{}
	
	// Map output meanings to seed words
	for meaning, activation := range activations {
		if activation > 0.3 {
			// Get words related to this meaning
			switch meaning {
			case "greeting":
				words = append(words, "hello", "hi", "greetings")
			case "assistance":
				words = append(words, "help", "assist", "support")
			case "technical":
				words = append(words, "code", "system", "process")
			case "problem":
				words = append(words, "error", "issue", "challenge")
			case "cognitive":
				words = append(words, "think", "understand", "analyze")
			case "comprehension":
				words = append(words, "see", "know", "grasp")
			}
		}
	}
	
	// Filter to only words in vocabulary
	vocabWords := []string{}
	vocab := brain.dataLoader.GetVocabulary()
	vocabMap := make(map[string]bool)
	for _, v := range vocab {
		vocabMap[v] = true
	}
	
	for _, word := range words {
		if vocabMap[word] {
			vocabWords = append(vocabWords, word)
		}
	}
	
	return vocabWords
}

// Legacy method
func (brain *LiquidStateBrain) selectNextWordByWaves(currentWord string, activations map[string]float64) string {
	// Try to get next word from transitions
	if nextWord, found := brain.dataLoader.GetNextWord(currentWord, 1.2); found {
		// Check if it aligns with current wave patterns
		for meaning, activation := range activations {
			if activation > 0.4 && brain.wordMatchesMeaning(nextWord, meaning) {
				return nextWord
			}
		}
		return nextWord
	}
	
	// Fallback: use wave patterns to guide selection
	return brain.dataLoader.GetStarterWord()
}

func (brain *LiquidStateBrain) wordMatchesMeaning(word, meaning string) bool {
	// Simple heuristic matching
	meaningWords := map[string][]string{
		"greeting":      {"hello", "hi", "hey", "greetings"},
		"assistance":    {"help", "assist", "support", "aid"},
		"technical":     {"code", "program", "system", "software"},
		"problem":       {"error", "bug", "issue", "problem"},
		"cognitive":     {"think", "thought", "mind", "brain"},
		"comprehension": {"understand", "know", "see", "grasp"},
	}
	
	if words, ok := meaningWords[meaning]; ok {
		for _, w := range words {
			if strings.Contains(word, w) || strings.Contains(w, word) {
				return true
			}
		}
	}
	
	return false
}

func (brain *LiquidStateBrain) simpleInterpretation(activations map[string]float64) string {
	// Find dominant activation
	maxActivation := 0.0
	dominantMeaning := "processing"
	
	for meaning, activation := range activations {
		if activation > maxActivation {
			maxActivation = activation
			dominantMeaning = meaning
		}
	}
	
	// Generate simple response based on dominant meaning
	switch dominantMeaning {
	case "greeting":
		return "Hello! The waves ripple with recognition."
	case "assistance":
		return "I sense you need help. Let the patterns guide us."
	case "technical":
		return "Technical waves detected. Processing computational patterns."
	case "problem":
		return "Error patterns emerging. Let's debug together."
	case "cognitive":
		return "Thought waves propagating through the reservoir."
	case "comprehension":
		return "Understanding crystallizes from the liquid patterns."
	default:
		return fmt.Sprintf("Wave patterns suggest: %s", dominantMeaning)
	}
}

func (brain *LiquidStateBrain) visualizeWaves() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	waveHistory := []WavePattern{}
	
	for {
		select {
		case <-brain.ctx.Done():
			return
		case wave := <-brain.wavePatterns:
			waveHistory = append(waveHistory, wave)
			
			// Keep only recent waves
			if len(waveHistory) > 1000 {
				waveHistory = waveHistory[100:]
			}
			
		case <-ticker.C:
			// Periodic visualization
			activeWaves := atomic.LoadInt64(&brain.activeWaves)
			if len(waveHistory) > 0 && activeWaves > 0 {
				brain.showWavePattern(waveHistory)
			}
		}
	}
}

func (brain *LiquidStateBrain) showWavePattern(waves []WavePattern) {
	// Create a 2D slice visualization (top view)
	grid := make([][]float64, brain.dimensions.X)
	for i := range grid {
		grid[i] = make([]float64, brain.dimensions.Y)
	}
	
	// Aggregate wave intensities
	for _, wave := range waves {
		age := time.Since(wave.timestamp).Seconds()
		if age < 1.0 { // Only show recent waves
			// Wave decays over time
			intensity := wave.intensity * math.Exp(-age*2)
			
			// Create ripple effect
			for dx := -3; dx <= 3; dx++ {
				for dy := -3; dy <= 3; dy++ {
					x, y := wave.origin[0]+dx, wave.origin[1]+dy
					if x >= 0 && x < brain.dimensions.X && 
					   y >= 0 && y < brain.dimensions.Y {
						distance := math.Sqrt(float64(dx*dx + dy*dy))
						grid[x][y] += intensity * math.Exp(-distance)
					}
				}
			}
		}
	}
	
	// Show wave visualization
	fmt.Println("\n🌊 Wave patterns in liquid reservoir:")
	for y := 0; y < min(10, brain.dimensions.Y); y++ {
		fmt.Print("   ")
		for x := 0; x < min(40, brain.dimensions.X); x++ {
			intensity := grid[x][y]
			if intensity > 0.8 {
				fmt.Print("●")
			} else if intensity > 0.5 {
				fmt.Print("◉")
			} else if intensity > 0.3 {
				fmt.Print("○")
			} else if intensity > 0.1 {
				fmt.Print("·")
			} else {
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}
}

// Individual neuron dynamics
func (n *LiquidNeuron) live() {
	ticker := time.NewTicker(time.Duration(5+rand.Intn(5)) * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			var state float64
			if val := n.state.Load(); val != nil {
				state = val.(float64)
			}
			
			// Check if neuron should fire
			if state > n.threshold && time.Since(n.lastFired).Milliseconds() > n.refractoryMs {
				// Fire!
				n.fire()
				n.lastFired = time.Now()
				
				// Reset state
				n.state.Store(0.1)
			} else {
				// Decay state
				n.state.Store(state * 0.95)
			}
			
			// Random spontaneous activity (keeps reservoir dynamic)
			if rand.Float64() < 0.001 {
				n.state.Store(state + 0.3)
			}
		}
	}
}

func (n *LiquidNeuron) fire() {
	// Send activation to all connected neurons
	for _, target := range n.connections {
		go func(t *LiquidNeuron) {
			// Synaptic delay
			time.Sleep(time.Duration(1+rand.Intn(3)) * time.Millisecond)
			
			// Activate target
			var current float64
			if val := t.state.Load(); val != nil {
				current = val.(float64)
			}
			// Random synaptic strength
			strength := 0.1 + rand.Float64()*0.4
			t.state.Store(math.Min(1.0, current+strength))
		}(target)
	}
}

func (o *OutputNeuron) monitor(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Calculate activation from connected neurons
			total := 0.0
			for _, neuron := range o.connections {
				if val := neuron.state.Load(); val != nil {
					total += val.(float64)
				}
			}
			
			o.activation.Store(total / float64(len(o.connections)))
		}
	}
}

func (brain *LiquidStateBrain) wordSimilarity(w1, w2 string) float64 {
	// Use dataset embeddings if available
	if brain.dataLoader != nil {
		return brain.dataLoader.ComputeSimilarity(w1, w2)
	}
	
	// Fallback to simple similarity
	if w1 == w2 {
		return 1.0
	}
	
	// Simple similarity for demo
	similarities := map[string]map[string]float64{
		"hello":      {"hi": 0.9, "greetings": 0.8, "hey": 0.85},
		"help":       {"assist": 0.9, "support": 0.8, "aid": 0.85},
		"code":       {"program": 0.9, "coding": 0.95, "software": 0.8},
		"error":      {"bug": 0.9, "issue": 0.8, "problem": 0.85},
		"think":      {"process": 0.8, "compute": 0.7, "consider": 0.85},
		"understand": {"comprehend": 0.9, "grasp": 0.8, "know": 0.7},
	}
	
	if score, ok := similarities[w1][w2]; ok {
		return score
	}
	if score, ok := similarities[w2][w1]; ok {
		return score
	}
	
	return 0.0
}

// max function is defined in utils.go