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
	// try to parse the response
	var resp map[string]any
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		if s.cfg.Debug {
			slog.Warn("[goutils.ai] failed to get json by calling json.Unmarshal, let's try to extract", "input", input, "error", err)
		}

		// use regex to extract the json part
		// it could be multiple lines.
		// this regex will find the smallest substring that starts with { and ends with }, capturing everything in between—even if it spans multiple lines.
		re := regexp.MustCompile(`(?s)\{.*\}`)
		input = re.FindString(input)
		// replace \\n -> \n
		input = regexp.MustCompile(`\\n`).ReplaceAllString(input, "\n")
		input = regexp.MustCompile(`\n`).ReplaceAllString(input, "")
		input = regexp.MustCompile(`\"`).ReplaceAllString(input, "\"")

		if err := json.Unmarshal([]byte(input), &resp); err != nil {
			if s.cfg.Debug {
				slog.Warn("[goutils.ai] failed to extract json", "input", input, "error", err)
			}

			// try to extract json from markdown
			if err := s.GrabJsonOutputFromMd(ctx, input, &resp); err != nil {
				if s.cfg.Debug {
					slog.Error("[goutils.ai] failed to extract json from md", "input", input, "error", err)
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
