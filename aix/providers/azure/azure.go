package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/lyricat/goutils/aix/chat"
)

type Config struct {
	APIKey     string
	Endpoint   string
	Deployment string
}

type Provider struct {
	client     *azopenai.Client
	deployment string
}

func New(cfg Config) (*Provider, error) {
	if cfg.APIKey == "" || cfg.Endpoint == "" {
		return nil, fmt.Errorf("azure openai api key and endpoint are required")
	}
	cred := azcore.NewKeyCredential(cfg.APIKey)
	client, err := azopenai.NewClientWithKeyCredential(cfg.Endpoint, cred, nil)
	if err != nil {
		return nil, err
	}
	return &Provider{
		client:     client,
		deployment: cfg.Deployment,
	}, nil
}

func (p *Provider) Chat(ctx context.Context, req *chat.Request) (*chat.Result, error) {
	messages := make([]azopenai.ChatRequestMessageClassification, 0, len(req.Messages))
	for _, m := range req.Messages {
		switch m.Role {
		case chat.RoleSystem:
			messages = append(messages, &azopenai.ChatRequestSystemMessage{
				Content: azopenai.NewChatRequestSystemMessageContent(m.Content),
			})
		case chat.RoleAssistant:
			msg := &azopenai.ChatRequestAssistantMessage{
				Content: azopenai.NewChatRequestAssistantMessageContent(m.Content),
			}
			if m.Name != "" {
				msg.Name = &m.Name
			}
			if len(m.ToolCalls) > 0 {
				msg.ToolCalls = make([]azopenai.ChatCompletionsToolCallClassification, 0, len(m.ToolCalls))
				for _, tc := range m.ToolCalls {
					toolType := "function"
					if tc.Type != "" {
						toolType = tc.Type
					}
					id := tc.ID
					fn := &azopenai.FunctionCall{
						Name:      &tc.Function.Name,
						Arguments: &tc.Function.Arguments,
					}
					msg.ToolCalls = append(msg.ToolCalls, &azopenai.ChatCompletionsFunctionToolCall{
						ID:       &id,
						Type:     &toolType,
						Function: fn,
					})
				}
			}
			messages = append(messages, msg)
		case chat.RoleTool:
			msg := &azopenai.ChatRequestToolMessage{
				Content: azopenai.NewChatRequestToolMessageContent(m.Content),
			}
			if m.ToolCallID != "" {
				msg.ToolCallID = &m.ToolCallID
			}
			messages = append(messages, msg)
		default:
			messages = append(messages, &azopenai.ChatRequestUserMessage{
				Content: azopenai.NewChatRequestUserMessageContent(m.Content),
			})
		}
	}

	payload := azopenai.ChatCompletionsOptions{
		Messages:       messages,
		DeploymentName: &p.deployment,
	}

	if req.Options.Temperature != nil {
		temp := float32(*req.Options.Temperature)
		payload.Temperature = &temp
	}
	if req.Options.TopP != nil {
		topP := float32(*req.Options.TopP)
		payload.TopP = &topP
	}
	if req.Options.MaxTokens != nil {
		maxTokens := int32(*req.Options.MaxTokens)
		payload.MaxTokens = &maxTokens
	}
	if len(req.Options.Stop) > 0 {
		payload.Stop = append([]string{}, req.Options.Stop...)
	}
	if req.Options.PresencePenalty != nil {
		val := float32(*req.Options.PresencePenalty)
		payload.PresencePenalty = &val
	}
	if req.Options.FrequencyPenalty != nil {
		val := float32(*req.Options.FrequencyPenalty)
		payload.FrequencyPenalty = &val
	}
	if req.Options.User != nil {
		payload.User = req.Options.User
	}

	if len(req.Tools) > 0 {
		payload.Tools = make([]azopenai.ChatCompletionsToolDefinitionClassification, 0, len(req.Tools))
		for _, tool := range req.Tools {
			if tool.Type != "function" {
				continue
			}
			name := tool.Function.Name
			fn := &azopenai.ChatCompletionsFunctionToolDefinitionFunction{
				Name:        &name,
				Description: nil,
				Parameters:  tool.Function.ParametersJSONSchema,
				Strict:      tool.Function.Strict,
			}
			if tool.Function.Description != "" {
				desc := tool.Function.Description
				fn.Description = &desc
			}
			toolType := "function"
			payload.Tools = append(payload.Tools, &azopenai.ChatCompletionsFunctionToolDefinition{
				Type:     &toolType,
				Function: fn,
			})
		}
	}

	if req.ToolChoice != nil {
		switch req.ToolChoice.Mode {
		case "auto":
			payload.ToolChoice = azopenai.ChatCompletionsToolChoiceAuto
		case "none":
			payload.ToolChoice = azopenai.ChatCompletionsToolChoiceNone
		case "required":
			// Azure SDK doesn't expose a direct helper for "required".
			// Fall back to auto to avoid invalid request construction.
			payload.ToolChoice = azopenai.ChatCompletionsToolChoiceAuto
		case "function":
			payload.ToolChoice = azopenai.NewChatCompletionsToolChoice(
				azopenai.ChatCompletionsToolChoiceFunction{Name: req.ToolChoice.FunctionName},
			)
		}
	}

	resp, err := p.client.GetChatCompletions(ctx, payload, nil)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message == nil {
		return &chat.Result{Raw: resp}, nil
	}

	text := ""
	for _, choice := range resp.Choices {
		if choice.Message != nil && choice.Message.Content != nil {
			text += *choice.Message.Content
		}
	}

	result := &chat.Result{
		Text:  text,
		Raw:   resp,
		Model: "",
	}
	if resp.Usage != nil {
		if resp.Usage.PromptTokens != nil {
			result.Usage.InputTokens = int(*resp.Usage.PromptTokens)
		}
		if resp.Usage.CompletionTokens != nil {
			result.Usage.OutputTokens = int(*resp.Usage.CompletionTokens)
		}
		if resp.Usage.TotalTokens != nil {
			result.Usage.TotalTokens = int(*resp.Usage.TotalTokens)
		} else {
			result.Usage.TotalTokens = result.Usage.InputTokens + result.Usage.OutputTokens
		}
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil && len(resp.Choices[0].Message.ToolCalls) > 0 {
		for _, tc := range resp.Choices[0].Message.ToolCalls {
			if fnCall, ok := tc.(*azopenai.ChatCompletionsFunctionToolCall); ok && fnCall.Function != nil {
				id := ""
				if fnCall.ID != nil {
					id = *fnCall.ID
				}
				toolType := ""
				if fnCall.Type != nil {
					toolType = *fnCall.Type
				}
				name := ""
				args := ""
				if fnCall.Function.Name != nil {
					name = *fnCall.Function.Name
				}
				if fnCall.Function.Arguments != nil {
					args = *fnCall.Function.Arguments
				}
				result.ToolCalls = append(result.ToolCalls, chat.ToolCall{
					ID:   id,
					Type: toolType,
					Function: chat.ToolCallFunction{
						Name:      name,
						Arguments: args,
					},
				})
			}
		}
	}

	return result, nil
}
