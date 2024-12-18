package bayesian

import (
	"encoding/gob"
	"fmt"
	"os"
	"sync"
)

type Explanation struct {
	Words                 []string
	WordSpamProb          map[string]float64 // P(word|Spam)
	WordHamProb           map[string]float64 // P(word|Ham)
	SpamPrior             float64            // P(Spam)
	HamPrior              float64            // P(Ham)
	SpamPosterior         float64            // P(Spam|Message)
	HamPosterior          float64            // P(Ham|Message)
	CombinedLikelihood    float64            // P(Words|Spam)
	CombinedHamLikelihood float64            // P(Words|Ham)
}

type Model struct {
	WordProbs      map[string]float64 // Probability of word appearing in spam (P(word|Spam))
	SpamCount      int                // Number of spam documents seen
	HamCount       int                // Number of ham (non-spam) documents seen
	WordSpamCounts map[string]int     // Count of documents containing word in spam
	WordHamCounts  map[string]int     // Count of documents containing word in ham
	mu             sync.RWMutex       // Mutex for thread safety
}

// NewModel creates a new, empty model.
func NewModel() *Model {
	return &Model{
		WordProbs:      make(map[string]float64),
		WordSpamCounts: make(map[string]int),
		WordHamCounts:  make(map[string]int),
	}
}

// Train trains the model with the given input. `input` is the tokenized message,
// and `isSpam` indicates if the message is spam. This updates internal counts and
// recalculates probabilities.
func (m *Model) Train(input []string, isSpam bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if isSpam {
		m.SpamCount++
	} else {
		m.HamCount++
	}

	// Use a set of words to ensure we only count each word once per document.
	wordSet := make(map[string]bool)
	for _, w := range input {
		if w == "" {
			continue
		}
		wordSet[w] = true
	}

	// Update counts
	for w := range wordSet {
		if isSpam {
			m.WordSpamCounts[w]++
		} else {
			m.WordHamCounts[w]++
		}
	}

	// Recalculate probabilities for words in this input.
	// P(word|Spam) = (WordSpamCount(word) + 1) / (SpamCount + 2)
	// We use +1 smoothing. The denominator +2 accounts for smoothing as well.
	for w := range wordSet {
		spamCount := m.WordSpamCounts[w]
		m.WordProbs[w] = float64(spamCount+1) / float64(m.SpamCount+2)
	}
}

// IsSpam classifies the given input and returns a boolean indicating spam/ham and the spam probability.
// Uses the formula:
// P(Spam|Message) = P(Spam)*Π(P(word|Spam)) / [ P(Spam)*Π(P(word|Spam)) + P(Ham)*Π(P(word|Ham)) ]
func (m *Model) IsSpam(input []string) (bool, float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.SpamCount == 0 && m.HamCount == 0 {
		// No training data
		return false, 0.5
	}

	pSpam := float64(m.SpamCount) / float64(m.SpamCount+m.HamCount)
	pHam := float64(m.HamCount) / float64(m.SpamCount+m.HamCount)

	// Compute likelihoods
	spamLikelihood := 1.0
	hamLikelihood := 1.0

	// Use a set to avoid counting word multiple times for doc probability
	wordSet := make(map[string]bool)
	for _, w := range input {
		if w == "" {
			continue
		}
		wordSet[w] = true
	}

	for w := range wordSet {
		pWordSpam := m.getWordSpamProb(w)
		pWordHam := m.getWordHamProb(w)

		spamLikelihood *= pWordSpam
		hamLikelihood *= pWordHam
	}

	// Apply Bayes' Theorem
	spamPosterior := (pSpam * spamLikelihood) / ((pSpam * spamLikelihood) + (pHam * hamLikelihood))

	return spamPosterior >= 0.5, spamPosterior
}

// Explain provides a detailed explanation of how the model computed the spam probability for the input.
func (m *Model) Explain(input []string) (*Explanation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.SpamCount+m.HamCount == 0 {
		return nil, fmt.Errorf("model is not trained yet")
	}

	pSpam := float64(m.SpamCount) / float64(m.SpamCount+m.HamCount)
	pHam := float64(m.HamCount) / float64(m.SpamCount+m.HamCount)

	wordSet := make(map[string]bool)
	for _, w := range input {
		if w == "" {
			continue
		}
		wordSet[w] = true
	}

	wordSpamProb := make(map[string]float64)
	wordHamProb := make(map[string]float64)

	spamLikelihood := 1.0
	hamLikelihood := 1.0

	for w := range wordSet {
		pWordSpam := m.getWordSpamProb(w)
		pWordHam := m.getWordHamProb(w)

		wordSpamProb[w] = pWordSpam
		wordHamProb[w] = pWordHam

		spamLikelihood *= pWordSpam
		hamLikelihood *= pWordHam
	}

	spamPosterior := (pSpam * spamLikelihood) / ((pSpam * spamLikelihood) + (pHam * hamLikelihood))
	hamPosterior := 1.0 - spamPosterior

	expl := &Explanation{
		Words:                 input,
		WordSpamProb:          wordSpamProb,
		WordHamProb:           wordHamProb,
		SpamPrior:             pSpam,
		HamPrior:              pHam,
		SpamPosterior:         spamPosterior,
		HamPosterior:          hamPosterior,
		CombinedLikelihood:    spamLikelihood,
		CombinedHamLikelihood: hamLikelihood,
	}
	return expl, nil
}

// SaveModel saves the model to a file using encoding/gob.
func (m *Model) SaveModel(filename string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := gob.NewEncoder(file)
	err = enc.Encode(m)
	if err != nil {
		return err
	}

	return nil
}

// LoadModel loads the model from a file using encoding/gob.
func LoadModel(filename string) (*Model, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	dec := gob.NewDecoder(file)
	var model Model
	if err := dec.Decode(&model); err != nil {
		return nil, err
	}
	return &model, nil
}

// getWordSpamProb computes P(word|Spam) using add-one smoothing.
func (m *Model) getWordSpamProb(word string) float64 {
	spamCount := m.WordSpamCounts[word]
	return float64(spamCount+1) / float64(m.SpamCount+2)
}

// getWordHamProb computes P(word|Ham) using add-one smoothing.
func (m *Model) getWordHamProb(word string) float64 {
	hamCount := m.WordHamCounts[word]
	return float64(hamCount+1) / float64(m.HamCount+2)
}
