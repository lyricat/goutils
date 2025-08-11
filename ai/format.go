package ai

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func (s *Instant) GrabJsonOutput(ctx context.Context, input string, outputKeys ...string) (map[string]any, error) {
	originalInput := input
	
	// try to parse the response
	var resp map[string]any
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		if s.cfg.Debug {
			slog.Warn("[goutils.ai] failed to get json by calling json.Unmarshal, let's try to extract", "input", input, "error", err)
		}

		// First try to extract from markdown if present
		if strings.Contains(input, "```json") || strings.Contains(input, "```") {
			// Look for markdown code blocks (with or without json label)
			markdownPattern := regexp.MustCompile(`(?s)` + "```" + `(?:json)?\s*\n?(.*?)(?:` + "```" + `|$)`)
			if matches := markdownPattern.FindStringSubmatch(input); len(matches) > 1 {
				input = strings.TrimSpace(matches[1])
				// Try to repair incomplete JSON
				input = s.attemptJSONRepair(input)
			}
		} else {
			// use regex to extract the json part
			// it could be multiple lines.
			// this regex will find the smallest substring that starts with { and ends with }, capturing everything in between—even if it spans multiple lines.
			re := regexp.MustCompile(`(?s)\{.*\}`)
			input = re.FindString(input)
		}
		
		// If still no JSON found, try to find it anywhere in the text
		if input == "" {
			// Look for JSON starting from the last { in the original input
			lastBrace := strings.LastIndex(originalInput, "{")
			if lastBrace != -1 {
				input = originalInput[lastBrace:]
				input = s.attemptJSONRepair(input)
			}
		}
		
		// replace \\n -> \n
		input = regexp.MustCompile(`\\n`).ReplaceAllString(input, "\n")
		input = regexp.MustCompile(`\n`).ReplaceAllString(input, "")
		input = regexp.MustCompile(`\"`).ReplaceAllString(input, "\"")

		if err := json.Unmarshal([]byte(input), &resp); err != nil {
			if s.cfg.Debug {
				slog.Warn("[goutils.ai] failed to extract json", "input", input, "error", err)
			}

			// try to extract json from markdown
			if err := s.GrabJsonOutputFromMd(ctx, originalInput, &resp); err != nil {
				if s.cfg.Debug {
					slog.Error("[goutils.ai] failed to extract json from md", "input", originalInput, "error", err)
				}
				return nil, err
			}
		}
	}

	if len(outputKeys) == 0 {
		return resp, nil
	}

	// check if the response is valid
	outputs := make(map[string]any)
	for _, outputKey := range outputKeys {
		if val, ok := resp[outputKey]; !ok || val == "" {
			return nil, nil
		}
		outputs[outputKey] = resp[outputKey]
	}

	return outputs, nil
}

func (s *Instant) GrabJsonOutputFromMd(ctx context.Context, input string, ptrOutput interface{}) error {
	input = strings.TrimSpace(input)

	if strings.Contains(input, "```json") {
		trimed, err := extractCodeFromMarkdown(input)
		if err != nil {
			if s.cfg.Debug {
				slog.Warn("[goutils.ai] failed to extract json from md", "error", err)
			}
		} else {
			input = trimed
		}
	}

	if err := json.Unmarshal([]byte(input), ptrOutput); err != nil {
		if s.cfg.Debug {
			slog.Warn("[goutils.ai] failed to unmarshal json from md", "input", input, "error", err)
		}
		return err
	}
	return nil
}

func (s *Instant) GrabYamlOutput(ctx context.Context, input string) (string, error) {
	respMap, respArray, err := grabYamlAsMapOrArray(input)
	if err != nil {
		if s.cfg.Debug {
			slog.Warn("[goutils.ai] grabYamlAsMapOrArray failed, let's try to extract from md", "input", input, "error", err)
		}

		// try to extract yaml from markdown
		if err := s.GrabYamlOutputFromMd(ctx, input, &respMap, &respArray); err != nil {
			if s.cfg.Debug {
				slog.Error("[goutils.ai] failed to extract yaml from md", "input", input, "error", err)
			}
			return "", err
		}
	}

	if respMap != nil {
		// convert map to yaml
		yamlBytes, err := yaml.Marshal(respMap)
		if err != nil {
			return "", err
		}
		return string(yamlBytes), nil

	} else if respArray != nil {
		// convert array to yaml
		yamlBytes, err := yaml.Marshal(respArray)
		if err != nil {
			return "", err
		}
		return string(yamlBytes), nil
	}

	return "", nil
}

func grabYamlAsMapOrArray(input string) (map[string]any, []any, error) {
	var respDict map[string]any
	var respArray []any
	if err := yaml.Unmarshal([]byte(input), &respDict); err != nil {
		if err := yaml.Unmarshal([]byte(input), &respArray); err != nil {
			return nil, nil, err
		}
		return nil, respArray, nil
	}
	return respDict, nil, nil
}

func (s *Instant) GrabYamlOutputFromMd(ctx context.Context, input string, mapOutput *map[string]any, arrayOutput *[]any) error {
	input = strings.TrimSpace(input)

	// Support ```yaml
	if strings.Contains(input, "```yaml") {
		trimmed, err := extractCodeFromMarkdown(input)
		if err != nil {
			if s.cfg.Debug {
				slog.Warn("[goutils.ai] failed to extract yaml from md", "error", err)
			}
			// Continue trying to parse even if extraction fails, maybe the block wasn't formatted correctly
		} else {
			input = trimmed
		}
	}

	// Perform YAML unmarshaling
	respMap, respArray, err := grabYamlAsMapOrArray(input)
	if err != nil {
		if s.cfg.Debug {
			slog.Warn("[goutils.ai] failed to unmarshal yaml from md", "input", input, "error", err)
		}
		return err
	}

	if respMap != nil {
		*mapOutput = respMap
	} else if respArray != nil {
		*arrayOutput = respArray
	}

	return nil
}

// attemptJSONRepair tries to fix common JSON formatting issues
func (s *Instant) attemptJSONRepair(input string) string {
	if input == "" {
		return input
	}
	
	// Remove trailing commas before closing braces/brackets
	input = regexp.MustCompile(`,\s*([}\]])`).ReplaceAllString(input, "$1")
	
	// Count braces and brackets to see if we need to add closing ones
	openBraces := strings.Count(input, "{")
	closeBraces := strings.Count(input, "}")
	openBrackets := strings.Count(input, "[")
	closeBrackets := strings.Count(input, "]")
	
	// Check for incomplete strings - count unescaped quotes
	quoteCount := 0
	escaped := false
	inString := false
	for i, ch := range input {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			quoteCount++
			inString = !inString
		}
		// If we're at the end and still in a string, close it
		if i == len(input)-1 && inString {
			input += "\""
			quoteCount++
		}
	}
	
	// Add missing closing braces
	for i := closeBraces; i < openBraces; i++ {
		// If we're in the middle of a string value, close it first
		if quoteCount%2 == 1 {
			input += "\""
		}
		input += "}"
	}
	
	// Add missing closing brackets
	for i := closeBrackets; i < openBrackets; i++ {
		if quoteCount%2 == 1 {
			input += "\""
		}
		input += "]"
	}
	
	return input
}
