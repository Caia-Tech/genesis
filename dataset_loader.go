package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DatasetLoader handles loading and processing text datasets
type DatasetLoader struct {
	dataPath    string
	vocabulary  map[string]int
	wordFreq    map[string]float64
	embeddings  map[string][]float64
	documents   []Document
	transitions map[string]map[string]float64 // word -> next word -> probability
	starters    map[string]float64            // words that start sentences
	enders      map[string]bool               // words that end sentences
	mu          sync.RWMutex
	maxVocabSize int
}

type Document struct {
	Path    string
	Content string
	Tokens  []string
}

type TrainingConfig struct {
	DatasetPaths    []string
	MaxVocabSize    int
	EmbeddingDim    int
	MinWordFreq     int
	MaxDocuments    int
}

func NewDatasetLoader(config TrainingConfig) (*DatasetLoader, error) {
	// Validate configuration
	if config.MaxVocabSize <= 0 {
		config.MaxVocabSize = 50000 // Default
	}
	if config.EmbeddingDim <= 0 {
		config.EmbeddingDim = 128 // Default
	}
	if config.MaxDocuments <= 0 {
		config.MaxDocuments = 1000 // Default
	}
	
	loader := &DatasetLoader{
		vocabulary:   make(map[string]int),
		wordFreq:     make(map[string]float64),
		embeddings:   make(map[string][]float64),
		documents:    make([]Document, 0, config.MaxDocuments), // Pre-allocate with capacity
		transitions:  make(map[string]map[string]float64),
		starters:     make(map[string]float64),
		enders:       make(map[string]bool),
		maxVocabSize: config.MaxVocabSize,
	}

	// Load all documents with progress tracking
	totalPaths := len(config.DatasetPaths)
	for i, path := range config.DatasetPaths {
		fmt.Printf("📚 Loading dataset %d/%d: %s\n", i+1, totalPaths, path)
		if err := loader.loadFromPath(path, config.MaxDocuments); err != nil {
			fmt.Printf("⚠️  Warning: failed to load dataset from %s: %v\n", path, err)
			continue // Continue with other datasets instead of failing completely
		}
	}
	
	// Check if any documents were loaded
	if len(loader.documents) == 0 {
		return nil, fmt.Errorf("no documents were successfully loaded from any dataset path")
	}

	// Build vocabulary and embeddings
	loader.buildVocabulary(config.MinWordFreq)
	loader.generateEmbeddings(config.EmbeddingDim)
	loader.buildTransitions()

	return loader, nil
}

func (dl *DatasetLoader) loadFromPath(path string, maxDocs int) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return dl.loadDirectory(path, maxDocs)
	}
	return dl.loadFile(path)
}

func (dl *DatasetLoader) loadDirectory(dirPath string, maxDocs int) error {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}

	docsLoaded := 0
	for _, file := range files {
		if docsLoaded >= maxDocs && maxDocs > 0 {
			break
		}

		if strings.HasSuffix(file.Name(), ".txt") || strings.HasSuffix(file.Name(), ".md") {
			fullPath := filepath.Join(dirPath, file.Name())
			if err := dl.loadFile(fullPath); err != nil {
				fmt.Printf("Warning: failed to load %s: %v\n", fullPath, err)
				continue
			}
			docsLoaded++
		}
	}

	return nil
}

func (dl *DatasetLoader) loadFile(filePath string) error {
	// Check file size first to prevent loading huge files
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("cannot stat file %s: %w", filePath, err)
	}
	
	// Limit file size to prevent memory issues (10MB limit)
	maxFileSize := int64(10 * 1024 * 1024)
	if fileInfo.Size() > maxFileSize {
		return fmt.Errorf("file %s is too large (%d bytes > %d bytes limit)", filePath, fileInfo.Size(), maxFileSize)
	}
	
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	
	// Validate content is not empty
	if len(content) == 0 {
		return fmt.Errorf("file %s is empty", filePath)
	}

	// Tokenize content
	tokens := dl.tokenize(string(content))
	
	doc := Document{
		Path:    filePath,
		Content: string(content),
		Tokens:  tokens,
	}

	dl.documents = append(dl.documents, doc)

	// Update word frequencies with limit check
	totalTokens := 0
	for _, token := range tokens {
		dl.wordFreq[token]++
		totalTokens++
		
		// Prevent excessive memory usage
		if len(dl.wordFreq) > dl.maxVocabSize*2 {
			fmt.Printf("⚠️  Word frequency map getting large (%d entries), consider reducing vocabulary\n", len(dl.wordFreq))
		}
	}
	
	fmt.Printf("✅ Loaded %s: %d tokens, %d unique words\n", filePath, totalTokens, len(tokens))

	return nil
}

func (dl *DatasetLoader) tokenize(text string) []string {
	// Simple tokenization - can be improved with better NLP libraries
	text = strings.ToLower(text)
	
	// Replace punctuation with spaces
	replacer := strings.NewReplacer(
		".", " ",
		",", " ",
		"!", " ",
		"?", " ",
		";", " ",
		":", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		"\"", " ",
		"'", " ",
		"\n", " ",
		"\t", " ",
	)
	text = replacer.Replace(text)

	// Split and filter
	words := strings.Fields(text)
	tokens := make([]string, 0, len(words))
	
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) > 1 { // Skip single characters
			tokens = append(tokens, word)
		}
	}

	return tokens
}

func (dl *DatasetLoader) buildVocabulary(minFreq int) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// Sort words by frequency and build vocabulary
	vocabIndex := 0
	for word, freq := range dl.wordFreq {
		if freq >= float64(minFreq) && vocabIndex < dl.maxVocabSize {
			dl.vocabulary[word] = vocabIndex
			vocabIndex++
		}
	}

	fmt.Printf("Built vocabulary with %d words\n", len(dl.vocabulary))
}

func (dl *DatasetLoader) generateEmbeddings(dim int) {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	
	// Validate embedding dimension
	if dim <= 0 || dim > 1024 {
		fmt.Printf("⚠️  Invalid embedding dimension %d, using 128\n", dim)
		dim = 128
	}
	
	fmt.Printf("🧮 Generating %d-dimensional embeddings for %d words...\n", dim, len(dl.vocabulary))

	// Generate embeddings based on word co-occurrence patterns
	cooccurrence := make(map[string]map[string]float64)
	windowSize := 5

	// Build co-occurrence matrix with memory limits
	maxCooccurrenceEntries := 100000 // Limit to prevent memory explosion
	cooccurrenceCount := 0
	
	for docIdx, doc := range dl.documents {
		// Progress indicator for large datasets
		if docIdx%100 == 0 && docIdx > 0 {
			fmt.Printf("⚡ Processing document %d/%d for embeddings\n", docIdx, len(dl.documents))
		}
		
		for i, word1 := range doc.Tokens {
			if _, exists := dl.vocabulary[word1]; !exists {
				continue
			}

			if cooccurrence[word1] == nil {
				cooccurrence[word1] = make(map[string]float64)
			}

			// Look at surrounding words
			start := max(0, i-windowSize)
			end := min(len(doc.Tokens), i+windowSize+1)

			for j := start; j < end; j++ {
				if i == j {
					continue
				}
				
				// Check memory limit
				if cooccurrenceCount >= maxCooccurrenceEntries {
					fmt.Printf("⚠️  Reached co-occurrence limit (%d), stopping early to prevent OOM\n", maxCooccurrenceEntries)
					goto embeddings_generation
				}
				
				word2 := doc.Tokens[j]
				if _, exists := dl.vocabulary[word2]; exists {
					distance := math.Abs(float64(i - j))
					cooccurrence[word1][word2] += 1.0 / distance
					cooccurrenceCount++
				}
			}
		}
	}
	
embeddings_generation:

	// Generate embeddings from co-occurrence patterns
	for word := range dl.vocabulary {
		embedding := make([]float64, dim)
		
		// Initialize with small random values
		for i := range embedding {
			embedding[i] = (rand.Float64() - 0.5) * 0.1
		}

		// Adjust based on co-occurrence
		if neighbors, exists := cooccurrence[word]; exists {
			for neighbor, weight := range neighbors {
				if nIdx, exists := dl.vocabulary[neighbor]; exists {
					// Simple embedding: use vocabulary index and weight
					embedding[nIdx%dim] += weight * 0.01
				}
			}
		}

		// Normalize
		norm := 0.0
		for _, val := range embedding {
			norm += val * val
		}
		norm = math.Sqrt(norm)
		if norm > 0 {
			for i := range embedding {
				embedding[i] /= norm
			}
		}

		dl.embeddings[word] = embedding
	}
}

func (dl *DatasetLoader) GetEmbedding(word string) ([]float64, bool) {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	embedding, exists := dl.embeddings[strings.ToLower(word)]
	return embedding, exists
}

func (dl *DatasetLoader) GetVocabulary() []string {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	words := make([]string, 0, len(dl.vocabulary))
	for word := range dl.vocabulary {
		words = append(words, word)
	}
	return words
}

func (dl *DatasetLoader) GetDocuments() []Document {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	return dl.documents
}

func (dl *DatasetLoader) ComputeSimilarity(word1, word2 string) float64 {
	emb1, exists1 := dl.GetEmbedding(word1)
	emb2, exists2 := dl.GetEmbedding(word2)

	if !exists1 || !exists2 {
		return 0.0
	}

	// Cosine similarity
	dot := 0.0
	for i := range emb1 {
		dot += emb1[i] * emb2[i]
	}

	return dot
}

// Training batch for neural models
type TrainingBatch struct {
	Inputs  [][]string
	Targets []string
}

func (dl *DatasetLoader) GenerateTrainingBatches(batchSize int, contextSize int) []TrainingBatch {
	dl.mu.RLock()
	defer dl.mu.RUnlock()

	batches := []TrainingBatch{}
	currentBatch := TrainingBatch{
		Inputs:  make([][]string, 0, batchSize),
		Targets: make([]string, 0, batchSize),
	}

	for _, doc := range dl.documents {
		tokens := doc.Tokens
		
		for i := contextSize; i < len(tokens)-1; i++ {
			// Use previous words as context
			context := make([]string, contextSize)
			for j := 0; j < contextSize; j++ {
				context[j] = tokens[i-contextSize+j]
			}

			target := tokens[i]

			currentBatch.Inputs = append(currentBatch.Inputs, context)
			currentBatch.Targets = append(currentBatch.Targets, target)

			if len(currentBatch.Inputs) >= batchSize {
				batches = append(batches, currentBatch)
				currentBatch = TrainingBatch{
					Inputs:  make([][]string, 0, batchSize),
					Targets: make([]string, 0, batchSize),
				}
			}
		}
	}

	// Add remaining batch
	if len(currentBatch.Inputs) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}

// Helper functions (only define if not already defined)

func (dl *DatasetLoader) buildTransitions() {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// Build word transition probabilities
	for _, doc := range dl.documents {
		tokens := doc.Tokens
		
		// Track sentence starters
		if len(tokens) > 0 {
			dl.starters[tokens[0]]++
		}
		
		// Build transitions
		for i := 0; i < len(tokens)-1; i++ {
			current := tokens[i]
			next := tokens[i+1]
			
			// Only track words in vocabulary
			if _, inVocab1 := dl.vocabulary[current]; !inVocab1 {
				continue
			}
			if _, inVocab2 := dl.vocabulary[next]; !inVocab2 {
				continue
			}
			
			if dl.transitions[current] == nil {
				dl.transitions[current] = make(map[string]float64)
			}
			dl.transitions[current][next]++
			
			// Track potential sentence enders
			if i == len(tokens)-2 || (i < len(tokens)-2 && isCapitalized(tokens[i+2])) {
				dl.enders[next] = true
			}
		}
	}
	
	// Normalize transition probabilities
	for word, transitions := range dl.transitions {
		total := 0.0
		for _, count := range transitions {
			total += count
		}
		if total > 0 {
			for nextWord, count := range transitions {
				dl.transitions[word][nextWord] = count / total
			}
		}
	}
	
	// Normalize starter probabilities
	totalStarters := 0.0
	for _, count := range dl.starters {
		totalStarters += count
	}
	if totalStarters > 0 {
		for word, count := range dl.starters {
			dl.starters[word] = count / totalStarters
		}
	}
	
	fmt.Printf("Built transitions for %d words\n", len(dl.transitions))
}

func isCapitalized(word string) bool {
	if len(word) == 0 {
		return false
	}
	return strings.ToUpper(word[:1]) == word[:1]
}

// GetNextWord returns a probable next word given the current word
func (dl *DatasetLoader) GetNextWord(currentWord string, temperature float64) (string, bool) {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	
	transitions, exists := dl.transitions[currentWord]
	if !exists || len(transitions) == 0 {
		return "", false
	}
	
	// Apply temperature to probabilities
	if temperature <= 0 {
		temperature = 1.0
	}
	
	// Find best next word (for now, using max probability)
	// TODO: Implement proper sampling with temperature
	var bestWord string
	bestProb := 0.0
	
	for word, prob := range transitions {
		// Apply temperature scaling
		scaledProb := math.Pow(prob, 1.0/temperature)
		if scaledProb > bestProb {
			bestProb = scaledProb
			bestWord = word
		}
	}
	
	return bestWord, true
}

// GetStarterWord returns a word that commonly starts sentences
func (dl *DatasetLoader) GetStarterWord() string {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	
	if len(dl.starters) == 0 {
		// Fallback to any word
		for word := range dl.vocabulary {
			return word
		}
		return "the"
	}
	
	// For now, return the most common starter
	var bestWord string
	bestProb := 0.0
	
	for word, prob := range dl.starters {
		if prob > bestProb {
			bestProb = prob
			bestWord = word
		}
	}
	
	return bestWord
}

// IsEnder checks if a word commonly ends sentences
func (dl *DatasetLoader) IsEnder(word string) bool {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	
	return dl.enders[word]
}

// GetTransitions returns the transition probabilities for a word
func (dl *DatasetLoader) GetTransitions(word string) (map[string]float64, bool) {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	
	transitions, exists := dl.transitions[word]
	if !exists {
		return nil, false
	}
	
	// Return a copy to avoid concurrent modification
	copy := make(map[string]float64)
	for k, v := range transitions {
		copy[k] = v
	}
	return copy, true
}