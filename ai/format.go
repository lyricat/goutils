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

func (s *Instant) GrabYamlOutput(ctx context.Context, input string, outputKeys ...string) (map[string]any, error) {
	var resp map[string]any
	if err := yaml.Unmarshal([]byte(input), &resp); err != nil {
		if s.cfg.Debug {
			slog.Warn("[goutils.ai] failed to get yaml by calling yaml.Unmarshal, let's try to extract from md", "input", input, "error", err)
		}

		// try to extract yaml from markdown
		if err := s.GrabYamlOutputFromMd(ctx, input, &resp); err != nil {
			if s.cfg.Debug {
				slog.Error("[goutils.ai] failed to extract yaml from md", "input", input, "error", err)
			}
			return nil, err
		}
	}

	if len(outputKeys) == 0 {
		return resp, nil
	}

	// check if the response is valid
	outputs := make(map[string]any)
	for _, outputKey := range outputKeys {
		if val, ok := resp[outputKey]; !ok || val == "" {
			// Consider if empty string is a valid value for YAML, unlike the JSON check
			// For now, mimicking the JSON logic.
			return nil, nil
		}
		outputs[outputKey] = resp[outputKey]
	}

	return outputs, nil
}

func (s *Instant) GrabYamlOutputFromMd(ctx context.Context, input string, ptrOutput interface{}) error {
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
	if err := yaml.Unmarshal([]byte(input), ptrOutput); err != nil {
		if s.cfg.Debug {
			slog.Warn("[goutils.ai] failed to unmarshal yaml from md", "input", input, "error", err)
		}
		return err
	}
	return nil
}
