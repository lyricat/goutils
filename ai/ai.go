package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/lyricat/goutils/ai/core"
	"gopkg.in/yaml.v3"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/aws/aws-sdk-go/aws"
	AwsCre "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/bedrockruntime"
	"github.com/aws/aws-sdk-go/service/bedrockruntime/bedrockruntimeiface"
	"github.com/sashabaranov/go-openai"
)

type Instant struct {
	cfg               core.Config
	openaiClient      *openai.Client
	azureOpenAIClient *azopenai.Client
	bedrockClient     bedrockruntimeiface.BedrockRuntimeAPI
}

func New(cfg core.Config) *Instant {
	var openaiClient *openai.Client
	var azureOpenAIClient *azopenai.Client
	var bedrockClient bedrockruntimeiface.BedrockRuntimeAPI
	var err error

	if IsOpenAICompatible(cfg.Provider) {
		openaiClient, err = createOpenAICompatibleClient(cfg)
		if err != nil {
			slog.Error("[goutils.ai] createOpenAICompatibleClient error", "error", err)
			return nil
		}
	}

	if cfg.AzureOpenAIAPIKey != "" && cfg.AzureOpenAIEndpoint != "" && cfg.AzureOpenAIModel != "" {
		keyCredential := azcore.NewKeyCredential(cfg.AzureOpenAIAPIKey)
		azureOpenAIClient, err = azopenai.NewClientWithKeyCredential(cfg.AzureOpenAIEndpoint, keyCredential, nil)
		if err != nil {
			slog.Error("[goutils.ai] NewClientWithKeyCredential error", "error", err)
			return nil
		}
	}

	if cfg.AwsBedrockModelArn != "" {
		sess := session.Must(session.NewSession((&aws.Config{
			Region: aws.String("us-east-1"),
			Credentials: AwsCre.NewStaticCredentials(
				cfg.AwsKey,    // id
				cfg.AwsSecret, // secret
				""),           // token can be left blank for now
		})))
		bedrockClient = bedrockruntime.New(sess)
	}

	return &Instant{
		cfg:               cfg,
		openaiClient:      openaiClient,
		azureOpenAIClient: azureOpenAIClient,
		bedrockClient:     bedrockClient,
	}
}

func (s *Instant) RawRequest(ctx context.Context, messages []core.Message) (*core.Result, error) {
	return s.RawRequestWithParams(ctx, messages, nil)
}

func (s *Instant) RawRequestWithParams(ctx context.Context, messages []core.Message, params map[string]any) (*core.Result, error) {
	if s.cfg.Debug {
		slog.Info("[goutils.ai] RawRequest messages:")
		for _, message := range messages {
			slog.Info("[goutils.ai] RawRequest message", "message", message.Pretty())
		}
	}

	var ret = &core.Result{}
	var err error

	_opts := &core.RawRequestOptions{}
	if params != nil {
		if val, ok := params["format"]; ok {
			if val == "json" {
				_opts.Format = core.FormatJSON
			} else if val == "yaml" {
				_opts.Format = core.FormatYAML
			}
		}
		if val, ok := params["model"]; ok {
			if val != nil && val != "" {
				_opts.Model = val.(string)
			}
		}
	}

	switch s.cfg.Provider {
	case core.ProviderOpenAI, core.ProviderXAI, core.ProviderDeepseek, core.ProviderGemini:
		_messages := make([]openai.ChatCompletionMessage, 0, len(messages))
		for _, message := range messages {
			_messages = append(_messages, openai.ChatCompletionMessage{
				Role:    message.Role,
				Content: message.Content,
			})
		}

		ret, err = s.OpenAIRawRequest(ctx, _messages, _opts)
		if err != nil {
			return ret, err
		}

	case core.ProviderAzure:
		_messages := make([]azopenai.ChatRequestMessageClassification, 0, len(messages))
		for _, message := range messages {
			if message.Role == openai.ChatMessageRoleUser {
				_messages = append(_messages, &azopenai.ChatRequestUserMessage{
					Content: azopenai.NewChatRequestUserMessageContent(message.Content),
				})
			} else if message.Role == openai.ChatMessageRoleAssistant {
				_messages = append(_messages, &azopenai.ChatRequestAssistantMessage{
					Content: azopenai.NewChatRequestAssistantMessageContent(message.Content),
				})
			}
		}

		ret, err = s.AzureOpenAIRawRequest(ctx, _messages, _opts)
		if err != nil {
			return ret, err
		}

	case core.ProviderBedrock:
		_messages := make([]BedRockClaudeChatMessage, 0, len(messages))
		for _, message := range messages {
			m := BedRockClaudeChatMessage{
				Role: message.Role,
				Content: []BedRockClaudeMessageContent{
					{
						Type: "text",
						Text: message.Content,
					},
				},
			}
			// @TODO: uncomment this when bedrock supports cache control for public use
			// if message.EnableCache {
			// 	m.Content[0].CacheControl = &BedRockClaudeCacheControl{
			// 		Type: "ephemeral",
			// 	}
			// }
			_messages = append(_messages, m)
		}
		ret, err = s.BedrockRawRequest(ctx, _messages, _opts)
		if err != nil {
			return ret, err
		}

	case core.ProviderAnthropic:
		_messages := make([]AnthropicChatMessage, 0, len(messages))
		for _, message := range messages {
			m := AnthropicChatMessage{
				Role: message.Role,
				Content: []AnthropicMessageContent{
					{
						Type: "text",
						Text: message.Content,
					},
				},
			}
			if message.EnableCache {
				m.Content[0].CacheControl = &AnthropicCacheControl{
					Type: "ephemeral",
				}
			}
			_messages = append(_messages, m)
		}
		ret, err = s.AnthropicRawRequest(ctx, _messages, _opts)

	case core.ProviderSusanoo:
		resp, err := s.SusanooRawRequest(ctx, messages, params)
		if err != nil {
			return nil, err
		}
		if _opts.Format == core.FormatJSON {
			ret.Json = resp.Data.Result
			buf, err := json.Marshal(ret.Json)
			if err != nil {
				ret.Text = fmt.Sprintf("%+v", ret.Json)
			}
			ret.Text = string(buf)
		} else if _opts.Format == core.FormatYAML {
			ret.Json = resp.Data.Result
			buf, err := yaml.Marshal(ret.Json)
			if err != nil {
				ret.Text = fmt.Sprintf("%+v", ret.Json)
			}
			ret.Text = string(buf)
		} else {
			if val, ok := resp.Data.Result["response"]; ok {
				ret.Text = val.(string)
			}
		}

	default:
		return nil, fmt.Errorf("provider %s not supported", s.cfg.Provider)
	}

	if err != nil {
		return nil, err
	}

	if s.cfg.Debug {
		slog.Info("[goutils.ai] RawRequest", "ret", ret)
	}

	return ret, nil
}

func (s *Instant) OneTimeRequestWithParams(ctx context.Context, content string, params map[string]any) (*core.Result, error) {
	resp, err := s.RawRequestWithParams(ctx, []core.Message{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: content,
		},
	}, params)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Instant) MultipleSteps(ctx context.Context, params core.ChainParams) (*core.Result, error) {
	newSteps := make([]core.ChainParamsStep, 0)
	for _, step := range params.Steps {
		if step.Instruction == "" && step.Input == "" {
			continue
		}
		if step.Input != "" {
			inst := fmt.Sprintf("Please read the following text and just say the word \"OK\". Do not explain the text: \n\n  %s", step.Input)
			newSteps = append(newSteps, core.ChainParamsStep{
				Options:     nil,
				Input:       "",
				Instruction: inst,
			})
		} else if step.Instruction != "" {
			newSteps = append(newSteps, core.ChainParamsStep{
				Options:     nil,
				Input:       "",
				Instruction: step.Instruction,
			})
		}
	}
	params.Steps = newSteps
	return s.CallInChain(ctx, params)
}

func (s *Instant) CallInChain(ctx context.Context, params core.ChainParams) (*core.Result, error) {
	ret := &core.Result{}
	conv := make([]core.Message, 0)
	for i := 0; i < len(params.Steps)-1; i++ {
		conv = append(conv, core.Message{
			Role:    openai.ChatMessageRoleUser,
			Content: params.Steps[i].Instruction,
		})

		resp, err := s.RawRequestWithParams(ctx, conv, params.RawRequestParams)
		if err != nil {
			return nil, err
		}

		conv = append(conv, core.Message{
			Role:    openai.ChatMessageRoleAssistant,
			Content: resp.Text,
		})
	}

	finalStep := params.Steps[len(params.Steps)-1]
	conv = append(conv, core.Message{
		Role:    openai.ChatMessageRoleUser,
		Content: finalStep.Instruction,
	})

	if params.RawRequestParams == nil {
		params.RawRequestParams = make(map[string]any)
	}
	if _, ok := params.RawRequestParams["format"]; !ok {
		params.RawRequestParams["format"] = params.Format
	}

	resp, err := s.RawRequestWithParams(ctx, conv, params.RawRequestParams)
	if err != nil {
		return nil, err
	}

	if params.Format == "json" {
		if len(resp.Json) == 0 {
			js, err := s.GrabJsonOutput(ctx, resp.Text)
			if err != nil {
				slog.Error("[goutils.ai] GrabJsonOutput error", "error", err)
				return nil, err
			}
			ret.Json = js
		}
	}

	ret.Text = resp.Text
	return ret, nil
}

func (s *Instant) GetEmbeddings(ctx context.Context, input []string) ([]float32, error) {
	switch s.cfg.Provider {
	case core.ProviderAzure:
		vec, err := s.CreateEmbeddingAzureOpenAI(ctx, input)
		if err != nil {
			slog.Error("[goutils.ai] CreateEmbeddingAzureOpenAI error", "error", err)
			return nil, err
		}
		return vec, nil
	case core.ProviderOpenAI:
		return s.CreateEmbeddingOpenAI(ctx, input)
	case core.ProviderBedrock:
		return s.CreateEmbeddingBedrock(ctx, input)
	case core.ProviderSusanoo:
		// @TODO replace with susanoo embedding
		vec, err := s.CreateEmbeddingAzureOpenAI(ctx, input)
		if err != nil {
			slog.Error("[goutils.ai] CreateEmbeddingAzureOpenAI error", "error", err)
			return nil, err
		}
		return vec, nil
	default:
		return nil, fmt.Errorf("provider %s not supported for embeddings", s.cfg.Provider)
	}
}
