package main

import (
	"math"
	"sort"
	"strings"
)

// ResponseGenerator handles advanced text generation with beam search
type ResponseGenerator struct {
	dataLoader      *DatasetLoader
	beamWidth       int
	maxLength       int
	temperature     float64
	topicMemory     map[string]float64
	contextWindow   []string
	grammarPatterns map[string][]string
}

// Beam represents a partial response being generated
type Beam struct {
	words      []string
	score      float64
	lastWord   string
	topicScore float64
	complete   bool
}

func NewResponseGenerator(dataLoader *DatasetLoader) *ResponseGenerator {
	gen := &ResponseGenerator{
		dataLoader:      dataLoader,
		beamWidth:       4,
		maxLength:       15, // Shorter responses
		temperature:     0.8,
		topicMemory:     make(map[string]float64),
		contextWindow:   make([]string, 0),
		grammarPatterns: initializeGrammarPatterns(),
	}
	
	return gen
}

func initializeGrammarPatterns() map[string][]string {
	return map[string][]string{
		"greeting_start": {"hello", "hi", "greetings", "hey"},
		"question_start": {"what", "how", "why", "when", "where", "who", "can", "do"},
		"statement_start": {"the", "i", "we", "you", "this", "that", "it"},
		"connectors": {"and", "but", "or", "so", "because", "however", "therefore"},
		"endings": {".", "!", "?"},
	}
}

// Generate creates a response using beam search
func (gen *ResponseGenerator) Generate(input string, activeConcepts []string) string {
	// Update context and topic memory
	gen.updateContext(input)
	gen.updateTopicMemory(activeConcepts)
	
	// Initialize beams with starter words
	beams := gen.initializeBeams(input, activeConcepts)
	
	// Beam search
	for step := 0; step < gen.maxLength && !gen.allBeamsComplete(beams); step++ {
		newBeams := []Beam{}
		
		for _, beam := range beams {
			if beam.complete {
				newBeams = append(newBeams, beam)
				continue
			}
			
			// Expand beam with possible next words
			expansions := gen.expandBeam(beam, activeConcepts)
			newBeams = append(newBeams, expansions...)
		}
		
		// Keep top beams
		beams = gen.selectTopBeams(newBeams)
	}
	
	// Select best complete response
	bestBeam := gen.selectBestResponse(beams)
	return gen.formatResponse(bestBeam)
}

func (gen *ResponseGenerator) updateContext(input string) {
	words := strings.Fields(strings.ToLower(input))
	gen.contextWindow = append(gen.contextWindow, words...)
	
	// Keep context window size limited
	if len(gen.contextWindow) > 50 {
		gen.contextWindow = gen.contextWindow[len(gen.contextWindow)-50:]
	}
}

func (gen *ResponseGenerator) updateTopicMemory(concepts []string) {
	// Decay existing topics
	for topic := range gen.topicMemory {
		gen.topicMemory[topic] *= 0.8
		if gen.topicMemory[topic] < 0.1 {
			delete(gen.topicMemory, topic)
		}
	}
	
	// Add new concepts
	for _, concept := range concepts {
		gen.topicMemory[concept] = 1.0
	}
}

func (gen *ResponseGenerator) initializeBeams(input string, activeConcepts []string) []Beam {
	beams := []Beam{}
	inputWords := strings.Fields(strings.ToLower(input))
	
	// Determine response type based on input
	responseType := gen.classifyInput(inputWords)
	
	// Get appropriate starter words
	starters := gen.getStarterWords(responseType, activeConcepts)
	
	for _, starter := range starters {
		beam := Beam{
			words:      []string{starter},
			score:      gen.scoreWord(starter, nil, activeConcepts),
			lastWord:   starter,
			topicScore: gen.calculateTopicRelevance(starter),
			complete:   false,
		}
		beams = append(beams, beam)
	}
	
	// Ensure we have at least one beam
	if len(beams) == 0 {
		beams = append(beams, Beam{
			words:    []string{gen.dataLoader.GetStarterWord()},
			lastWord: gen.dataLoader.GetStarterWord(),
			score:    1.0,
		})
	}
	
	return beams
}

func (gen *ResponseGenerator) classifyInput(words []string) string {
	if len(words) == 0 {
		return "statement"
	}
	
	firstWord := words[0]
	
	// Check for question words
	questionWords := []string{"what", "how", "why", "when", "where", "who", "can", "do", "is", "are"}
	for _, qw := range questionWords {
		if firstWord == qw {
			return "question"
		}
	}
	
	// Check for greetings
	greetings := []string{"hello", "hi", "hey", "greetings"}
	for _, g := range greetings {
		if firstWord == g {
			return "greeting"
		}
	}
	
	return "statement"
}

func (gen *ResponseGenerator) getStarterWords(responseType string, activeConcepts []string) []string {
	starters := []string{}
	
	switch responseType {
	case "greeting":
		starters = append(starters, "hello", "hi", "greetings")
	case "question":
		// For questions, we might start with affirmative or explanation words
		starters = append(starters, "yes", "i", "the", "this", "that")
	default:
		// For statements, use a mix of common starters
		starters = append(starters, "i", "the", "this", "we", "that")
	}
	
	// Add some activated concepts as potential starters
	for i, concept := range activeConcepts {
		if i < 2 { // Limit to avoid too many options
			starters = append(starters, concept)
		}
	}
	
	// Filter to only words in vocabulary
	validStarters := []string{}
	vocab := gen.dataLoader.GetVocabulary()
	vocabMap := make(map[string]bool)
	for _, v := range vocab {
		vocabMap[v] = true
	}
	
	for _, starter := range starters {
		if vocabMap[starter] {
			validStarters = append(validStarters, starter)
		}
	}
	
	return validStarters
}

func (gen *ResponseGenerator) expandBeam(beam Beam, activeConcepts []string) []Beam {
	expansions := []Beam{}
	
	// Get transition candidates
	transitions, exists := gen.dataLoader.GetTransitions(beam.lastWord)
	if !exists || len(transitions) == 0 {
		// If no transitions, try to end the sentence gracefully
		beam.complete = true
		return []Beam{beam}
	}
	
	// Score and rank candidates
	candidates := gen.rankCandidates(transitions, beam, activeConcepts)
	
	// Take top candidates
	for i, candidate := range candidates {
		if i >= gen.beamWidth {
			break
		}
		
		newBeam := Beam{
			words:      append(append([]string{}, beam.words...), candidate.word),
			score:      beam.score + candidate.score,
			lastWord:   candidate.word,
			topicScore: beam.topicScore + gen.calculateTopicRelevance(candidate.word),
			complete:   gen.shouldComplete(beam, candidate.word),
		}
		
		expansions = append(expansions, newBeam)
	}
	
	return expansions
}

type wordCandidate struct {
	word  string
	score float64
}

func (gen *ResponseGenerator) rankCandidates(transitions map[string]float64, beam Beam, activeConcepts []string) []wordCandidate {
	candidates := []wordCandidate{}
	
	// Count word frequencies in current response and recent context
	wordCounts := make(map[string]int)
	for _, w := range beam.words {
		wordCounts[w]++
	}
	
	// Also count recent context to avoid repetition
	for _, w := range gen.contextWindow {
		wordCounts[w]++
	}
	
	for word, prob := range transitions {
		// Skip if word is used too much recently
		if wordCounts[word] > 1 {
			continue
		}
		
		// Skip very short words unless they're important
		if len(word) < 3 && !gen.isImportantWord(word) {
			continue
		}
		
		score := gen.scoreWord(word, &beam, activeConcepts) * prob
		candidates = append(candidates, wordCandidate{word, score})
	}
	
	// Sort by score
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})
	
	return candidates
}

func (gen *ResponseGenerator) isImportantWord(word string) bool {
	importantWords := []string{"i", "you", "we", "is", "are", "can", "do", "to", "of", "in", "on", "at"}
	for _, iw := range importantWords {
		if word == iw {
			return true
		}
	}
	return false
}

func (gen *ResponseGenerator) scoreWord(word string, beam *Beam, activeConcepts []string) float64 {
	score := 1.0
	
	// Topic relevance
	if topicScore, exists := gen.topicMemory[word]; exists {
		score *= (1.0 + topicScore)
	}
	
	// Concept activation bonus
	for _, concept := range activeConcepts {
		if word == concept {
			score *= 2.0
		} else if strings.Contains(word, concept) || strings.Contains(concept, word) {
			score *= 1.3
		}
	}
	
	// Grammar coherence
	if beam != nil && len(beam.words) > 0 {
		lastWord := beam.words[len(beam.words)-1]
		
		// Strongly penalize immediate repetition
		if word == lastWord {
			score *= 0.1
		}
		
		// Check for recent repetition
		for i := len(beam.words) - 1; i >= max(0, len(beam.words)-3); i-- {
			if beam.words[i] == word {
				score *= 0.3
			}
		}
		
		// Bonus for good word combinations
		if gen.isGoodTransition(lastWord, word) {
			score *= 1.4
		}
	}
	
	// Word quality bonus
	if len(word) >= 4 {
		score *= 1.1 // Prefer longer, more meaningful words
	}
	
	// Penalize very common words unless they're grammatically important
	commonWords := []string{"and", "the", "of", "to", "a", "in", "that", "is"}
	for _, cw := range commonWords {
		if word == cw && beam != nil && len(beam.words) < 3 {
			score *= 0.8
		}
	}
	
	return score
}

// max function moved to utils.go to avoid duplicates

func (gen *ResponseGenerator) isGoodTransition(word1, word2 string) bool {
	// Simple grammar rules
	goodTransitions := map[string][]string{
		"the":  {"best", "most", "first", "last", "only"},
		"is":   {"a", "the", "very", "quite", "not"},
		"can":  {"help", "assist", "be", "you", "we"},
		"i":    {"can", "will", "think", "understand", "believe"},
		"you":  {"can", "will", "are", "have", "need"},
		"and":  {"the", "i", "you", "we", "then"},
		"but":  {"i", "you", "we", "the", "not"},
	}
	
	if words, ok := goodTransitions[word1]; ok {
		for _, w := range words {
			if w == word2 {
				return true
			}
		}
	}
	
	return false
}

func (gen *ResponseGenerator) calculateTopicRelevance(word string) float64 {
	relevance := 0.0
	
	for topic, weight := range gen.topicMemory {
		similarity := gen.wordSimilarity(word, topic)
		relevance += similarity * weight
	}
	
	return relevance
}

func (gen *ResponseGenerator) wordSimilarity(word1, word2 string) float64 {
	if word1 == word2 {
		return 1.0
	}
	
	// Check embeddings
	emb1, exists1 := gen.dataLoader.GetEmbedding(word1)
	emb2, exists2 := gen.dataLoader.GetEmbedding(word2)
	
	if exists1 && exists2 {
		// Cosine similarity
		dot := 0.0
		for i := range emb1 {
			dot += emb1[i] * emb2[i]
		}
		return dot
	}
	
	// Simple substring matching
	if strings.Contains(word1, word2) || strings.Contains(word2, word1) {
		return 0.5
	}
	
	return 0.0
}

func (gen *ResponseGenerator) shouldComplete(beam Beam, nextWord string) bool {
	// Check if we've reached a natural ending
	if gen.dataLoader.IsEnder(nextWord) {
		return true
	}
	
	// Check length
	if len(beam.words) >= gen.maxLength-1 {
		return true
	}
	
	// Check for ending punctuation
	if nextWord == "." || nextWord == "!" || nextWord == "?" {
		return true
	}
	
	return false
}

func (gen *ResponseGenerator) selectTopBeams(beams []Beam) []Beam {
	// Sort by score
	sort.Slice(beams, func(i, j int) bool {
		return beams[i].score > beams[j].score
	})
	
	// Keep top beams
	if len(beams) > gen.beamWidth*2 {
		beams = beams[:gen.beamWidth*2]
	}
	
	return beams
}

func (gen *ResponseGenerator) allBeamsComplete(beams []Beam) bool {
	for _, beam := range beams {
		if !beam.complete {
			return false
		}
	}
	return true
}

func (gen *ResponseGenerator) selectBestResponse(beams []Beam) Beam {
	if len(beams) == 0 {
		return Beam{words: []string{"I", "understand"}}
	}
	
	// Score beams by multiple criteria
	bestBeam := beams[0]
	bestScore := gen.scoreResponse(beams[0])
	
	for _, beam := range beams[1:] {
		score := gen.scoreResponse(beam)
		if score > bestScore {
			bestScore = score
			bestBeam = beam
		}
	}
	
	return bestBeam
}

func (gen *ResponseGenerator) scoreResponse(beam Beam) float64 {
	if len(beam.words) == 0 {
		return 0.0
	}
	
	// Base score
	score := beam.score
	
	// Length penalty (prefer medium length)
	idealLength := 10.0
	lengthDiff := math.Abs(float64(len(beam.words)) - idealLength)
	score *= math.Exp(-lengthDiff * 0.1)
	
	// Topic coherence bonus
	score *= (1.0 + beam.topicScore*0.1)
	
	// Diversity bonus
	uniqueWords := make(map[string]bool)
	for _, word := range beam.words {
		uniqueWords[word] = true
	}
	diversityRatio := float64(len(uniqueWords)) / float64(len(beam.words))
	score *= (0.5 + diversityRatio*0.5)
	
	return score
}

func (gen *ResponseGenerator) formatResponse(beam Beam) string {
	if len(beam.words) == 0 {
		return "I need to process that."
	}
	
	words := beam.words
	
	// Capitalize first word
	if len(words) > 0 {
		words[0] = strings.Title(words[0])
	}
	
	// Join words
	response := strings.Join(words, " ")
	
	// Add ending punctuation if needed
	lastChar := response[len(response)-1:]
	if lastChar != "." && lastChar != "!" && lastChar != "?" {
		// Determine punctuation based on content
		if strings.Contains(response, "?") || gen.isQuestion(words) {
			response += "?"
		} else if len(words) < 5 {
			response += "."
		} else {
			response += "."
		}
	}
	
	return response
}

func (gen *ResponseGenerator) isQuestion(words []string) bool {
	if len(words) == 0 {
		return false
	}
	
	questionStarters := []string{"what", "how", "why", "when", "where", "who", "can", "do", "is", "are"}
	firstWord := strings.ToLower(words[0])
	
	for _, qs := range questionStarters {
		if firstWord == qs {
			return true
		}
	}
	
	return false
}