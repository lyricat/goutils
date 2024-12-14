package bayesian

import (
	"os"
	"testing"
)

func TestNewModel(t *testing.T) {
	model := NewModel()
	if model == nil {
		t.Error("NewModel() returned nil")
	}
	if model.WordProbs == nil {
		t.Error("WordProbs map not initialized")
	}
}

func TestTrainAndIsSpam(t *testing.T) {
	model := NewModel()

	// Train with spam examples
	spamInputs := [][]rune{
		[]rune("$$$"),
		[]rune("WIN"),
		[]rune("CASH"),
	}
	for _, input := range spamInputs {
		model.Train(input, true)
	}

	// Train with ham examples
	hamInputs := [][]rune{
		[]rune("hello"),
		[]rune("meeting"),
		[]rune("report"),
	}
	for _, input := range hamInputs {
		model.Train(input, false)
	}

	// Test spam detection
	testCases := []struct {
		input    []rune
		wantSpam bool
	}{
		{[]rune("$WIN$"), true},
		{[]rune("hello meeting"), false},
	}

	for _, tc := range testCases {
		isSpam, prob := model.IsSpam(tc.input)
		if isSpam != tc.wantSpam {
			t.Errorf("IsSpam(%v) = %v, want %v (probability: %v)", string(tc.input), isSpam, tc.wantSpam, prob)
		}
		if prob < 0.0 || prob > 1.0 {
			t.Errorf("Invalid probability %v for input %v", prob, string(tc.input))
		}
	}
}

func TestSaveAndLoadModel(t *testing.T) {
	model := NewModel()

	// Train the model
	model.Train([]rune("$$$"), true)
	model.Train([]rune("hello"), false)

	// Save the model
	tmpFile := "test_model.gob"
	defer os.Remove(tmpFile) // Clean up after test

	err := model.SaveModel(tmpFile)
	if err != nil {
		t.Fatalf("Failed to save model: %v", err)
	}

	// Load the model
	loadedModel, err := LoadModel(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	// Verify loaded model has same data
	if len(loadedModel.WordProbs) != len(model.WordProbs) {
		t.Error("Loaded model has different number of word probabilities")
	}
	if loadedModel.SpamCount != model.SpamCount {
		t.Error("Loaded model has different spam count")
	}
	if loadedModel.HamCount != model.HamCount {
		t.Error("Loaded model has different ham count")
	}

	// Verify loaded model produces same results
	input := []rune("test$$$")
	origIsSpam, origProb := model.IsSpam(input)
	loadedIsSpam, loadedProb := loadedModel.IsSpam(input)

	if origIsSpam != loadedIsSpam || origProb != loadedProb {
		t.Errorf("Loaded model produces different results: orig(%v, %v) != loaded(%v, %v)",
			origIsSpam, origProb, loadedIsSpam, loadedProb)
	}
}
