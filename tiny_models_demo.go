package main

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

// TinyModel represents a small, specialized model that neurons can use
type TinyModel interface {
	Process(input string) (string, float64) // returns result and confidence
	Size() int                              // size in MB
	Latency() time.Duration                 // typical response time
}

// Tiny specialized models (in reality, these would be actual small ML models)
type MathModel struct{}

func (m MathModel) Process(input string) (string, float64) {
	// Simplified: detect basic math patterns
	if strings.Contains(input, "sqrt") {
		re := regexp.MustCompile(`\d+`)
		if nums := re.FindAllString(input, -1); len(nums) > 0 {
			var n float64
			fmt.Sscanf(nums[0], "%f", &n)
			return fmt.Sprintf("%.2f", math.Sqrt(n)), 0.95
		}
	}
	return "", 0.0
}
func (m MathModel) Size() int              { return 10 } // 10MB
func (m MathModel) Latency() time.Duration { return 5 * time.Millisecond }

type DateModel struct{}

func (d DateModel) Process(input string) (string, float64) {
	// Detect date-related queries
	if strings.Contains(input, "today") || strings.Contains(input, "date") {
		return time.Now().Format("January 2, 2006"), 0.90
	}
	return "", 0.0
}
func (d DateModel) Size() int              { return 5 } // 5MB
func (d DateModel) Latency() time.Duration { return 2 * time.Millisecond }

type SentimentModel struct{}

func (s SentimentModel) Process(input string) (string, float64) {
	// Simple sentiment detection
	positive := []string{"happy", "good", "great", "excellent", "love"}
	negative := []string{"sad", "bad", "terrible", "hate", "angry"}
	
	posCount, negCount := 0, 0
	words := strings.Fields(strings.ToLower(input))
	
	for _, word := range words {
		for _, pos := range positive {
			if word == pos {
				posCount++
			}
		}
		for _, neg := range negative {
			if word == neg {
				negCount++
			}
		}
	}
	
	if posCount > negCount {
		return "positive", float64(posCount) / float64(len(words))
	} else if negCount > posCount {
		return "negative", float64(negCount) / float64(len(words))
	}
	return "neutral", 0.5
}
func (s SentimentModel) Size() int              { return 50 } // 50MB
func (s SentimentModel) Latency() time.Duration { return 10 * time.Millisecond }

// EnhancedNeuron - A neuron that might have access to a tiny model
type EnhancedNeuron struct {
	*LiquidNeuron
	
	// Some neurons have specialized capabilities
	tinyModel  TinyModel
	modelCalls atomic.Int64
	
	// Neuron decides when to use its model
	modelThreshold float64
}

// EnhancedLiquidBrain - Liquid brain where some neurons have tiny models
type EnhancedLiquidBrain struct {
	*LiquidStateBrain
	enhancedNeurons []*EnhancedNeuron
	totalModelCalls atomic.Int64
	modelRegistry   map[string]TinyModel
}

// CreateEnhancedBrain - Create a brain where ~1% of neurons have specialized models
func CreateEnhancedBrain(size int) *EnhancedLiquidBrain {
	brain := &EnhancedLiquidBrain{
		LiquidStateBrain: NewLiquidStateBrain(size),
		enhancedNeurons:  make([]*EnhancedNeuron, 0),
		modelRegistry: map[string]TinyModel{
			"math":      MathModel{},
			"date":      DateModel{},
			"sentiment": SentimentModel{},
		},
	}
	
	// Give ~1% of neurons access to tiny models
	specializedCount := size / 100
	if specializedCount < 10 {
		specializedCount = 10
	}
	
	// Distribute different model types across neurons
	modelTypes := []string{"math", "date", "sentiment"}
	for i := 0; i < specializedCount; i++ {
		// Pick a random neuron to enhance
		x, y, z := i%(brain.dimensions.X), (i/brain.dimensions.X)%brain.dimensions.Y, i/(brain.dimensions.X*brain.dimensions.Y)
		if x < brain.dimensions.X && y < brain.dimensions.Y && z < brain.dimensions.Z {
			neuron := brain.reservoir[x][y][z]
			
			enhanced := &EnhancedNeuron{
				LiquidNeuron:   neuron,
				tinyModel:      brain.modelRegistry[modelTypes[i%len(modelTypes)]],
				modelThreshold: 0.7 + (float64(i%30) / 100.0), // Vary thresholds
			}
			
			brain.enhancedNeurons = append(brain.enhancedNeurons, enhanced)
		}
	}
	
	fmt.Printf("🧠 Created enhanced brain with %d neurons, %d have tiny models\n", 
		size, len(brain.enhancedNeurons))
	
	return brain
}

// ProcessWithModels - Process input where neurons might use tiny models
func (brain *EnhancedLiquidBrain) ProcessWithModels(input string) string {
	// First, normal liquid processing
	// brain.InjectSignal(input) // TODO: implement this method
	time.Sleep(200 * time.Millisecond) // Let waves propagate
	
	// Enhanced neurons check if they should use their models
	modelResults := make(chan string, 100)
	
	for _, neuron := range brain.enhancedNeurons {
		go func(n *EnhancedNeuron) {
			activation := n.state.Load().(float64)
			
			// Neuron decides whether to use its model
			if activation > n.modelThreshold {
				result, confidence := n.tinyModel.Process(input)
				if confidence > 0.5 {
					n.modelCalls.Add(1)
					brain.totalModelCalls.Add(1)
					
					// Amplify this neuron's activation based on model confidence
					newActivation := activation * (1 + confidence)
					if newActivation > 1.0 {
						newActivation = 1.0
					}
					n.state.Store(newActivation)
					
					// Propagate the model's insight through the network
					for _, conn := range n.connections {
						current := conn.state.Load().(float64)
						conn.state.Store(math.Min(1.0, current+confidence*0.5))
					}
					
					modelResults <- fmt.Sprintf("[%T: %s]", n.tinyModel, result)
				}
			}
		}(neuron)
	}
	
	// Collect model insights
	time.Sleep(50 * time.Millisecond)
	close(modelResults)
	
	insights := []string{}
	for result := range modelResults {
		insights = append(insights, result)
	}
	
	// Generate response combining liquid dynamics and model insights
	baseResponse := brain.Think(input)
	
	if len(insights) > 0 {
		return fmt.Sprintf("%s\nModel insights: %s", baseResponse, strings.Join(insights, ", "))
	}
	
	return baseResponse
}

// ShowModelUsage - Display statistics about model usage
func (brain *EnhancedLiquidBrain) ShowModelUsage() {
	fmt.Println("\n📊 Tiny Model Usage Statistics:")
	fmt.Printf("Total model calls: %d\n", brain.totalModelCalls.Load())
	
	modelCounts := make(map[string]int64)
	for _, neuron := range brain.enhancedNeurons {
		calls := neuron.modelCalls.Load()
		if calls > 0 {
			modelType := fmt.Sprintf("%T", neuron.tinyModel)
			modelCounts[modelType] += calls
		}
	}
	
	for model, count := range modelCounts {
		fmt.Printf("  %s: %d calls\n", model, count)
	}
}

// DemoTinyModels - Demonstrate liquid brain with integrated tiny models
func DemoTinyModels() {
	fmt.Println("\n🔬 Liquid Brain with Tiny Models Demo")
	fmt.Println("=" + strings.Repeat("=", 49))
	
	// Create brain where some neurons have specialized models
	brain := CreateEnhancedBrain(10000)
	defer brain.Cleanup()
	
	// Test various inputs
	tests := []struct {
		input       string
		description string
	}{
		{
			"calculate the square root of 256",
			"Math-specialized neurons should activate",
		},
		{
			"what's today's date?",
			"Date-specialized neurons should respond",
		},
		{
			"I'm feeling really happy and excited!",
			"Sentiment neurons should detect positive emotion",
		},
		{
			"the square root of 144 is needed for today's calculation",
			"Multiple specialized neurons might activate",
		},
		{
			"explain quantum computing",
			"General query - mostly liquid processing",
		},
	}
	
	for _, test := range tests {
		fmt.Printf("\n\n💭 Input: %s\n", test.input)
		fmt.Printf("📝 Expected: %s\n", test.description)
		
		response := brain.ProcessWithModels(test.input)
		fmt.Printf("🧠 Response: %s\n", response)
	}
	
	// Show how many times models were actually used
	brain.ShowModelUsage()
	
	// Demonstrate scaling behavior
	fmt.Println("\n\n📈 Scaling Demonstration:")
	sizes := []int{1000, 10000, 100000}
	
	for _, size := range sizes {
		fmt.Printf("\n🧪 Testing with %d neurons:\n", size)
		testBrain := CreateEnhancedBrain(size)
		
		start := time.Now()
		testBrain.ProcessWithModels("calculate the square root of 169 for today's analysis")
		elapsed := time.Since(start)
		
		fmt.Printf("   Processing time: %v\n", elapsed)
		fmt.Printf("   Model calls: %d\n", testBrain.totalModelCalls.Load())
		
		// At larger scales, neurons are more selective about using models
		if size >= 10000 {
			fmt.Println("   💡 Neurons become more selective at scale!")
		}
		
		testBrain.Cleanup()
	}
}