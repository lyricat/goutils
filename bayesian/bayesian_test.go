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
	spamInputs := []string{
		"$$$",
		"WIN",
		"CASH",
	}
	model.Train(spamInputs, true)

	// Train with ham examples
	hamInputs := []string{
		"hello",
		"meeting",
		"report",
	}
	model.Train(hamInputs, false)

	// Test spam detection
	testCases := []struct {
		input    []string
		wantSpam bool
	}{
		{[]string{"WIN", "THE", "REWARDS!"}, true},
		{[]string{"hello", "meeting"}, false},
	}

	for _, tc := range testCases {
		isSpam, prob := model.IsSpam(tc.input)
		if isSpam != tc.wantSpam {
			t.Errorf("IsSpam(%v) = %v, want %v (probability: %v)", tc.input, isSpam, tc.wantSpam, prob)
		}
		if prob < 0.0 || prob > 1.0 {
			t.Errorf("Invalid probability %v for input %v", prob, tc.input)
		}
	}
}

func TestSaveAndLoadModel(t *testing.T) {
	model := NewModel()

	// Train the model
	model.Train([]string{"$$$"}, true)
	model.Train([]string{"hello"}, false)

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
	input := []string{"test $$$"}
	origIsSpam, origProb := model.IsSpam(input)
	loadedIsSpam, loadedProb := loadedModel.IsSpam(input)

	if origIsSpam != loadedIsSpam || origProb != loadedProb {
		t.Errorf("Loaded model produces different results: orig(%v, %v) != loaded(%v, %v)",
			origIsSpam, origProb, loadedIsSpam, loadedProb)
	}
}

func TestExplain(t *testing.T) {
	model := NewModel()

	// Train with spam examples
	spamInputs := []string{
		"$$$",
		"WIN",
		"CASH",
	}
	model.Train(spamInputs, true)

	// Train with ham examples
	hamInputs := []string{
		"hello",
		"meeting",
		"report",
	}
	model.Train(hamInputs, false)

	// Test explanation for mixed input (containing both spam and ham characteristics)
	input := []string{"hello", "$$$", "meeting"}
	explanation, err := model.Explain(input)
	if err != nil {
		t.Fatalf("Explain(%v) returned error: %v", input, err)
	}

	// Verify basic explanation properties
	if explanation.Probability <= 0.0 || explanation.Probability > 1.0 {
		t.Errorf("Invalid probability %v", explanation.Probability)
	}
	if explanation.PriorSpamProb <= 0.0 || explanation.PriorSpamProb > 1.0 {
		t.Errorf("Invalid prior spam probability %v", explanation.PriorSpamProb)
	}
	if explanation.PriorHamProb <= 0.0 || explanation.PriorHamProb > 1.0 {
		t.Errorf("Invalid prior ham probability %v", explanation.PriorHamProb)
	}

	// Verify character details
	if len(explanation.Details) == 0 {
		t.Error("No character details provided in explanation")
	}
	for _, detail := range explanation.Details {
		if detail.SpamProbability < 0.0 || detail.SpamProbability > 1.0 {
			t.Errorf("Invalid spam probability %v for character %v", detail.SpamProbability, detail.Char)
		}
	}

	// Verify top indicators
	if len(explanation.TopSpamIndicators) == 0 {
		t.Error("No top spam indicators provided")
	}
	if len(explanation.TopHamIndicators) == 0 {
		t.Error("No top ham indicators provided")
	}

	// Test explanation with no training data
	emptyModel := NewModel()
	emptyExplanation, err := emptyModel.Explain(input)
	if err != nil {
		t.Fatalf("Explain on empty model returned error: %v", err)
	}
	if emptyExplanation.Probability != 0.5 {
		t.Errorf("Expected neutral probability (0.5) for empty model, got %v", emptyExplanation.Probability)
	}
	if len(emptyExplanation.Details) != 0 {
		t.Error("Expected no details for empty model")
	}
}
