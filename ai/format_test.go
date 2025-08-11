package ai

import (
	"context"
	"reflect"
	"testing"

	"github.com/lyricat/goutils/ai/core"
)

func TestGrabJsonOutput(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name       string
		input      string
		outputKeys []string
		expected   map[string]any
		expectNil  bool
		debug      bool
	}{
		{
			name:  "Valid JSON without output keys",
			input: `{"name": "John", "age": 30, "city": "New York"}`,
			expected: map[string]any{
				"name": "John",
				"age":  float64(30),
				"city": "New York",
			},
			expectNil: false,
		},
		{
			name:       "Valid JSON with matching output keys",
			input:      `{"name": "John", "age": 30, "city": "New York"}`,
			outputKeys: []string{"name", "age"},
			expected: map[string]any{
				"name": "John",
				"age":  float64(30),
			},
			expectNil: false,
		},
		{
			name:       "Valid JSON with missing output key",
			input:      `{"name": "John", "age": 30}`,
			outputKeys: []string{"name", "age", "city"},
			expected:   nil,
			expectNil:  true,
		},
		{
			name:       "Valid JSON with empty value for required key",
			input:      `{"name": "John", "age": ""}`,
			outputKeys: []string{"name", "age"},
			expected:   nil,
			expectNil:  true,
		},
		{
			name:  "JSON embedded in text with regex extraction",
			input: `The response is: {"status": "success", "data": {"id": 123, "message": "Hello"}} and that's it.`,
			expected: map[string]any{
				"status": "success",
				"data": map[string]any{
					"id":      float64(123),
					"message": "Hello",
				},
			},
			expectNil: false,
			debug:     true,
		},
		{
			name:      "JSON with escaped characters",
			input:     `Here is the data: {\"name\": \"John\", \"message\": \"Hello\\nWorld\"}`,
			expected:  nil,
			expectNil: true,
			debug:     true,
		},
		{
			name: "Multiline JSON embedded in text",
			input: `The configuration is:
{
	"server": "localhost",
	"port": 8080,
	"ssl": true
}
End of configuration.`,
			expected: map[string]any{
				"server": "localhost",
				"port":   float64(8080),
				"ssl":    true,
			},
			expectNil: false,
		},
		{
			name:  "JSON in markdown code block",
			input: "Here is the JSON:\n```json\n{\"result\": \"success\", \"count\": 42}\n```\nThat's all.",
			expected: map[string]any{
				"result": "success",
				"count":  float64(42),
			},
			expectNil: false,
			debug:     true,
		},
		{
			name:  "Complex nested JSON",
			input: `{"users": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}], "total": 2}`,
			expected: map[string]any{
				"users": []any{
					map[string]any{"id": float64(1), "name": "Alice"},
					map[string]any{"id": float64(2), "name": "Bob"},
				},
				"total": float64(2),
			},
			expectNil: false,
		},
		{
			name:       "Empty JSON object",
			input:      `{}`,
			outputKeys: []string{"key"},
			expected:   nil,
			expectNil:  true,
		},
		{
			name:      "Invalid JSON fallback to markdown extraction",
			input:     "This is not valid JSON at all",
			expected:  nil,
			expectNil: true,
			debug:     true,
		},
		{
			name:  "JSON with special characters in strings",
			input: `{"path": "/home/user/file.txt", "regex": "^[a-z]+$"}`,
			expected: map[string]any{
				"path":  "/home/user/file.txt",
				"regex": "^[a-z]+$",
			},
			expectNil: false,
		},
		{
			name:  "JSON with null values",
			input: `{"name": "John", "age": null, "active": true}`,
			expected: map[string]any{
				"name":   "John",
				"age":    nil,
				"active": true,
			},
			expectNil: false,
		},
		{
			name:       "Output keys with null value should pass",
			input:      `{"id": 123, "description": null}`,
			outputKeys: []string{"id", "description"},
			expected: map[string]any{
				"id":          float64(123),
				"description": nil,
			},
			expectNil: false,
		},
		{
			name:  "JSON with literal \\n characters",
			input: `{"message": "Line 1\\nLine 2\\nLine 3"}`,
			expected: map[string]any{
				"message": "Line 1\\nLine 2\\nLine 3",
			},
			expectNil: false,
		},
		{
			name:  "JSON extraction with cleaning of escape sequences",
			input: `The AI responded with: {"answer": "The result is \"42\"", "confidence": 0.95}`,
			expected: map[string]any{
				"answer":     `The result is "42"`,
				"confidence": float64(0.95),
			},
			expectNil: false,
		},
		{
			name:       "JSON with boolean values and output keys",
			input:      `{"success": true, "error": false, "message": "Operation completed"}`,
			outputKeys: []string{"success", "error"},
			expected: map[string]any{
				"success": true,
				"error":   false,
			},
			expectNil: false,
		},
		{
			name:       "JSON in a markdown code block",
			input:      "```json\n{\"success\": true, \"message\": \"Operation completed\"}\n```",
			outputKeys: []string{"success", "message"},
			expected: map[string]any{
				"success": true,
				"message": "Operation completed",
			},
			expectNil: false,
		},
		{
			name:       "JSON in a broken markdown code block",
			input:      "```json\n{\"success\": true, \"message\": \"Operation completed\"}",
			outputKeys: []string{"success", "message"},
			expected: map[string]any{
				"success": true,
				"message": "Operation completed",
			},
			expectNil: false,
		},
		{
			name:       "JSON in a broken markdown code block and broken json",
			input:      "```json\n{\"success\": true, \"message\": \"Operation 'completed'",
			outputKeys: []string{"success", "message"},
			expected: map[string]any{
				"success": true,
				"message": "Operation 'completed'",
			},
			expectNil: false,
		},
		{
			name:       "JSON in a broken markdown code block and some reasoning text",
			input:      "<think>\n\n**Clarifying User Intent**\n\nI'm tement, \nfor delivery.\n\n\n\n</think>```json\n{\"success\": true, \"message\": \"Operation completed\"}",
			outputKeys: []string{"success", "message"},
			expected: map[string]any{
				"success": true,
				"message": "Operation completed",
			},
			expectNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create an Instant with the appropriate debug setting
			instant := &Instant{
				cfg: core.Config{
					Debug: tc.debug,
				},
			}

			result, err := instant.GrabJsonOutput(ctx, tc.input, tc.outputKeys...)

			if tc.expectNil {
				if err == nil && result != nil {
					t.Errorf("Expected nil result or error, but got result: %v", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !reflect.DeepEqual(result, tc.expected) {
					t.Errorf("Expected result:\n%+v\nGot result:\n%+v", tc.expected, result)
				}
			}
		})
	}
}

func TestGrabJsonOutput_EdgeCases(t *testing.T) {
	ctx := context.Background()
	instant := &Instant{
		cfg: core.Config{
			Debug: false,
		},
	}

	testCases := []struct {
		name       string
		input      string
		outputKeys []string
		shouldFail bool
	}{
		{
			name:       "Array instead of object",
			input:      `[1, 2, 3]`,
			outputKeys: []string{},
			shouldFail: true,
		},
		{
			name:       "Multiple JSON objects in text",
			input:      `First: {"a": 1} Second: {"b": 2}`,
			outputKeys: []string{},
			shouldFail: true, // Will fail because of text after first JSON
		},
		{
			name:       "Deeply nested JSON",
			input:      `{"level1": {"level2": {"level3": {"level4": {"value": "deep"}}}}}`,
			outputKeys: []string{},
			shouldFail: false,
		},
		{
			name:       "JSON with Unicode characters",
			input:      `{"emoji": "😀", "chinese": "你好", "arabic": "مرحبا"}`,
			outputKeys: []string{},
			shouldFail: false,
		},
		{
			name:       "Empty input string",
			input:      "",
			outputKeys: []string{},
			shouldFail: true,
		},
		{
			name:       "Only whitespace",
			input:      "   \n\t  ",
			outputKeys: []string{},
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := instant.GrabJsonOutput(ctx, tc.input, tc.outputKeys...)

			if tc.shouldFail {
				if err == nil && result != nil {
					t.Errorf("Expected failure but got result: %v", result)
				}
			} else {
				if err != nil && result == nil {
					t.Errorf("Expected success but got error: %v", err)
				}
			}
		})
	}
}
