package bayesian

import (
	"encoding/gob"
	"math"
	"os"
	"sync"
)

// Model represents a Bayesian filter model for spam detection
type Model struct {
	WordProbs map[string]float64 // Probability of word appearing in spam
	SpamCount int                // Number of spam documents seen
	HamCount  int                // Number of ham (non-spam) documents seen
	mu        sync.RWMutex       // Mutex for thread safety
}

// NewModel creates a new Bayesian filter model
func NewModel() *Model {
	return &Model{
		WordProbs: make(map[string]float64),
	}
}

// Train trains the model with the given input and classification
func (m *Model) Train(input []rune, isSpam bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert runes to words (using runes as individual tokens for char array input)
	words := make(map[string]bool)
	for _, r := range input {
		words[string(r)] = true
	}

	// Update counts
	if isSpam {
		m.SpamCount++
	} else {
		m.HamCount++
	}

	// Update word probabilities
	for word := range words {
		if isSpam {
			m.WordProbs[word] = (m.WordProbs[word]*float64(m.SpamCount-1) + 1) / float64(m.SpamCount)
		} else {
			m.WordProbs[word] = (m.WordProbs[word] * float64(m.SpamCount)) / float64(m.SpamCount+1)
		}
	}
}

// IsSpam checks if the input is spam and returns the probability
func (m *Model) IsSpam(input []rune) (bool, float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If we haven't seen any training data, return neutral probability
	if m.SpamCount == 0 || m.HamCount == 0 {
		return false, 0.5
	}

	// Calculate prior probabilities
	spamProb := math.Log(float64(m.SpamCount) / float64(m.SpamCount+m.HamCount))
	hamProb := math.Log(float64(m.HamCount) / float64(m.SpamCount+m.HamCount))

	// Calculate likelihood for each character
	for _, r := range input {
		word := string(r)
		if prob, exists := m.WordProbs[word]; exists {
			// Use Laplace smoothing to avoid zero probabilities
			smoothedSpamProb := (prob + 1) / (float64(m.SpamCount) + 2)
			smoothedHamProb := ((1 - prob) + 1) / (float64(m.HamCount) + 2)

			spamProb += math.Log(smoothedSpamProb)
			hamProb += math.Log(smoothedHamProb)
		}
	}

	// Convert log probabilities back to probabilities
	spamExp := math.Exp(spamProb)
	hamExp := math.Exp(hamProb)
	probability := spamExp / (spamExp + hamExp)

	return probability > 0.5, probability
}

// SaveModel saves the trained model to a file
func (m *Model) SaveModel(filename string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(m)
}

// LoadModel loads a trained model from a file
func LoadModel(filename string) (*Model, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var model Model
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&model); err != nil {
		return nil, err
	}

	return &model, nil
}
