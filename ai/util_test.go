package ai

import (
	"testing"
)

func TestExtractCodeFromMarkdown(t *testing.T) {
	testCases := []struct {
		name           string
		markdownInput  string
		expectedOutput string
		expectError    bool // Although the current function doesn't return errors, keep for future flexibility
	}{
		{
			name: "No code block",
			markdownInput: `This is just plain text.
No code here.`,
			expectedOutput: "",
			expectError:    false,
		},
		{
			name: "Single JSON code block",
			markdownInput: `Here is some JSON:
` + "```" + `json
{
  "key": "value"
}
` + "```" + `
More text.`,
			expectedOutput: `{
  "key": "value"
}`,
			expectError: false,
		},
		{
			name: "Single YAML code block",
			markdownInput: `Here is some YAML:
` + "```" + `yaml
key:
  - item1
  - item2
` + "```" + `
End of text.`,
			expectedOutput: `key:
  - item1
  - item2`,
			expectError: false,
		},
		{
			name: "Code block with other markdown elements",
			markdownInput: `# Heading
Some *italic* and **bold** text.
` + "```" + `json
[1, 2, 3]
` + "```" + `
> A blockquote`,
			expectedOutput: `[1, 2, 3]`,
			expectError:    false,
		},
		{
			name: "Empty JSON code block",
			markdownInput: `Empty block:
` + "```" + `json

` + "```" + `
`,
			expectedOutput: ``, // The function extracts the lines between ```, which is empty here.
			expectError:    false,
		},
		{
			name:           "Empty input string",
			markdownInput:  "",
			expectedOutput: "",
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualOutput, err := extractCodeFromMarkdown(tc.markdownInput)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
				// Optional: Check for specific error type or message if needed
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if actualOutput != tc.expectedOutput {
					t.Errorf("Expected output:\n%q\nGot output:\n%q", tc.expectedOutput, actualOutput)
				}
			}
		})
	}
}
