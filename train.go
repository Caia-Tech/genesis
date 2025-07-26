package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// TrainingMetrics tracks model performance
type TrainingMetrics struct {
	Perplexity     float64
	Accuracy       float64
	ResponseTime   time.Duration
	TotalExamples  int
	CorrectOutputs int
	mu             sync.RWMutex
}

func (tm *TrainingMetrics) Update(correct bool, responseTime time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.TotalExamples++
	if correct {
		tm.CorrectOutputs++
	}
	tm.Accuracy = float64(tm.CorrectOutputs) / float64(tm.TotalExamples)
	tm.ResponseTime = responseTime
}

func (tm *TrainingMetrics) String() string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	return fmt.Sprintf("Accuracy: %.2f%% | Examples: %d | Avg Response: %v",
		tm.Accuracy*100, tm.TotalExamples, tm.ResponseTime)
}

// ModelTrainer handles training for different model types
type ModelTrainer struct {
	config         *Config
	transparentLLM *TransparentLLM
	liquidBrain    *LiquidStateBrain
	dataLoader     *DatasetLoader
	metrics        *TrainingMetrics
	stopChan       chan struct{}
}

func NewModelTrainer(configPath string) (*ModelTrainer, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	dataLoader, err := NewDatasetLoader(config.Training)
	if err != nil {
		return nil, fmt.Errorf("failed to load datasets: %w", err)
	}

	trainer := &ModelTrainer{
		config:     config,
		dataLoader: dataLoader,
		metrics:    &TrainingMetrics{},
		stopChan:   make(chan struct{}),
	}

	// Initialize the selected model
	switch config.Model.Type {
	case "transparent":
		trainer.transparentLLM = NewTransparentLLMWithConfig(config)
	case "liquid":
		trainer.liquidBrain = NewLiquidStateBrainWithConfig(30, config) // 30x30x15 brain
	default:
		return nil, fmt.Errorf("unknown model type: %s", config.Model.Type)
	}

	return trainer, nil
}

func (mt *ModelTrainer) Train(epochs int) error {
	fmt.Printf("Starting training for %d epochs...\n", epochs)
	fmt.Printf("Model type: %s\n", mt.config.Model.Type)
	fmt.Printf("Vocabulary size: %d\n", len(mt.dataLoader.GetVocabulary()))
	fmt.Println(strings.Repeat("-", 50))

	// Generate training batches
	batches := mt.dataLoader.GenerateTrainingBatches(32, 5) // batch_size=32, context_size=5
	
	for epoch := 1; epoch <= epochs; epoch++ {
		select {
		case <-mt.stopChan:
			fmt.Println("\nTraining interrupted")
			return nil
		default:
			mt.runEpoch(epoch, batches)
		}
	}

	return nil
}

func (mt *ModelTrainer) runEpoch(epoch int, batches []TrainingBatch) {
	epochStart := time.Now()
	correctPredictions := 0
	totalPredictions := 0

	fmt.Printf("\nEpoch %d:\n", epoch)

	for i, batch := range batches {
		if i%10 == 0 {
			fmt.Printf("  Batch %d/%d - %s\n", i+1, len(batches), mt.metrics)
		}

		for j, context := range batch.Inputs {
			target := batch.Targets[j]
			
			// Process based on model type
			var predicted string
			var responseTime time.Duration
			
			switch mt.config.Model.Type {
			case "transparent":
				predicted, responseTime = mt.evaluateTransparent(context, target)
			case "liquid":
				predicted, responseTime = mt.evaluateLiquid(context, target)
			}

			correct := predicted == target
			mt.metrics.Update(correct, responseTime)
			
			if correct {
				correctPredictions++
			}
			totalPredictions++
		}
	}

	epochAccuracy := float64(correctPredictions) / float64(totalPredictions)
	epochDuration := time.Since(epochStart)
	
	fmt.Printf("  Epoch %d complete - Accuracy: %.2f%% - Duration: %v\n",
		epoch, epochAccuracy*100, epochDuration)
}

func (mt *ModelTrainer) evaluateTransparent(context []string, target string) (string, time.Duration) {
	start := time.Now()
	
	// Create input from context
	input := strings.Join(context, " ")
	
	// Get response
	response, thoughtChan := mt.transparentLLM.Understand(input)
	
	// Drain thought channel
	go func() {
		for range thoughtChan {
		}
	}()
	
	// For now, return whether the model activated the target concept
	targetNeuron := mt.transparentLLM.concepts[target]
	if targetNeuron != nil && targetNeuron.getActivation() > 0.5 {
		return target, time.Since(start)
	}
	
	return response, time.Since(start)
}

func (mt *ModelTrainer) evaluateLiquid(context []string, target string) (string, time.Duration) {
	start := time.Now()
	
	// Create input from context
	input := strings.Join(context, " ")
	
	// Get response
	response := mt.liquidBrain.Think(input)
	
	// Check if response contains target word
	if strings.Contains(strings.ToLower(response), target) {
		return target, time.Since(start)
	}
	
	return response, time.Since(start)
}

func (mt *ModelTrainer) InteractiveTest() {
	fmt.Println("\nEntering interactive test mode...")
	fmt.Println("Type 'quit' to exit")
	fmt.Println(strings.Repeat("-", 50))

	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Print("\n> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			continue
		}
		
		input = strings.TrimSpace(input)
		if input == "quit" || input == "exit" {
			break
		}

		start := time.Now()
		
		switch mt.config.Model.Type {
		case "transparent":
			response, thoughtChan := mt.transparentLLM.Understand(input)
			
			// Show thought process
			fmt.Println("\nThought process:")
			for range thoughtChan {
				// Already visualized in the model
			}
			
			fmt.Printf("\nResponse: %s\n", response)
			
		case "liquid":
			response := mt.liquidBrain.Think(input)
			fmt.Printf("\nResponse: %s\n", response)
		}
		
		fmt.Printf("Response time: %v\n", time.Since(start))
	}
}

func (mt *ModelTrainer) Cleanup() {
	close(mt.stopChan)
	
	if mt.transparentLLM != nil {
		mt.transparentLLM.Cleanup()
	}
	if mt.liquidBrain != nil {
		mt.liquidBrain.Cleanup()
	}
}

// Main training entry point
func TrainMain() {
	var (
		configPath string
		epochs     int
		testMode   bool
	)

	flag.StringVar(&configPath, "config", "config.json", "Path to configuration file")
	flag.IntVar(&epochs, "epochs", 10, "Number of training epochs")
	flag.BoolVar(&testMode, "test", false, "Run in interactive test mode")
	flag.Parse()

	// Create trainer
	trainer, err := NewModelTrainer(configPath)
	if err != nil {
		log.Fatalf("Failed to create trainer: %v", err)
	}
	defer trainer.Cleanup()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\nShutting down gracefully...")
		trainer.Cleanup()
		os.Exit(0)
	}()

	// Run training or test mode
	if testMode {
		trainer.InteractiveTest()
	} else {
		if err := trainer.Train(epochs); err != nil {
			log.Fatalf("Training failed: %v", err)
		}
		
		// Show final metrics
		fmt.Printf("\nFinal metrics: %s\n", trainer.metrics)
		
		// Optional: run interactive test after training
		fmt.Println("\nTraining complete. Starting interactive test mode...")
		trainer.InteractiveTest()
	}
}

// Evaluation metrics
func calculatePerplexity(predictions []float64) float64 {
	if len(predictions) == 0 {
		return math.Inf(1)
	}
	
	logSum := 0.0
	for _, p := range predictions {
		if p > 0 {
			logSum += math.Log(p)
		}
	}
	
	return math.Exp(-logSum / float64(len(predictions)))
}