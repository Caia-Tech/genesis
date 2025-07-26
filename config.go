package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// Config holds all configuration for the LLM system
type Config struct {
	Model        ModelConfig        `json:"model"`
	Training     TrainingConfig     `json:"training"`
	Resources    ResourceLimits     `json:"resources"`
	Datasets     DatasetConfig      `json:"datasets"`
}

type ModelConfig struct {
	Type           string `json:"type"` // "transparent", "liquid", "evolving"
	EmbeddingDim   int    `json:"embedding_dim"`
	HiddenSize     int    `json:"hidden_size"`
	NumLayers      int    `json:"num_layers"`
	MaxConcepts    int    `json:"max_concepts"`
}

type ResourceLimits struct {
	MaxGoroutines    int `json:"max_goroutines"`
	MaxMemoryMB      int `json:"max_memory_mb"`
	MaxNeurons       int `json:"max_neurons"`
	ChannelBufferSize int `json:"channel_buffer_size"`
}

type DatasetConfig struct {
	Paths            []string `json:"paths"`
	MaxDocuments     int      `json:"max_documents"`
	MinWordFrequency int      `json:"min_word_frequency"`
	TestSplitRatio   float64  `json:"test_split_ratio"`
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		Model: ModelConfig{
			Type:         "transparent",
			EmbeddingDim: 128,
			HiddenSize:   256,
			NumLayers:    3,
			MaxConcepts:  10000,
		},
		Training: TrainingConfig{
			DatasetPaths: []string{
				"datasets/conversational_corpus.txt",
				"datasets/high_quality_corpus.txt",
				"datasets/dialogue_patterns.txt",
			},
			MaxVocabSize: 50000,
			EmbeddingDim: 128,
			MinWordFreq:  2,
			MaxDocuments: 1000,
		},
		Resources: ResourceLimits{
			MaxGoroutines:     1000,
			MaxMemoryMB:       4096,
			MaxNeurons:        100000,
			ChannelBufferSize: 100,
		},
		Datasets: DatasetConfig{
			Paths: []string{
				"datasets/conversational_corpus.txt",
				"datasets/high_quality_corpus.txt",
				"datasets/dialogue_patterns.txt",
			},
			MaxDocuments:     1000,
			MinWordFrequency: 2,  // Reduced for testing
			TestSplitRatio:   0.2,
		},
	}
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create default config file
		defaultConfig := DefaultConfig()
		if err := SaveConfig(path, defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		fmt.Printf("Created default config at %s\n", path)
		return defaultConfig, nil
	}

	// Load existing config
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to a JSON file
func SaveConfig(path string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Model.EmbeddingDim <= 0 {
		return fmt.Errorf("embedding_dim must be positive")
	}
	if c.Model.HiddenSize <= 0 {
		return fmt.Errorf("hidden_size must be positive")
	}
	if c.Resources.MaxGoroutines <= 0 {
		return fmt.Errorf("max_goroutines must be positive")
	}
	if c.Resources.MaxMemoryMB <= 0 {
		return fmt.Errorf("max_memory_mb must be positive")
	}
	if len(c.Datasets.Paths) == 0 {
		return fmt.Errorf("at least one dataset path is required")
	}
	if c.Datasets.TestSplitRatio < 0 || c.Datasets.TestSplitRatio > 1 {
		return fmt.Errorf("test_split_ratio must be between 0 and 1")
	}
	return nil
}