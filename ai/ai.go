package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/aws/aws-sdk-go/aws"
	AwsCre "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/bedrockruntime"
	"github.com/aws/aws-sdk-go/service/bedrockruntime/bedrockruntimeiface"
	"github.com/sashabaranov/go-openai"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type (
	ChainParamsStep struct {
		Input       string
		Instruction string
		Options     any
	}

	ChainParams struct {
		Format string
		Steps  []ChainParamsStep
	}

	ChainResult struct {
		Text string
		Json map[string]any
	}

	Instant struct {
		cfg               Config
		openaiClient      *openai.Client
		azureOpenAIClient *azopenai.Client
		bedrockClient     bedrockruntimeiface.BedrockRuntimeAPI
	}

	Config struct {
		// openai
		OpenAIApiKey         string
		OpenAIGptModel       string
		OpenAIEmbeddingModel string

		// azure openai
		AzureOpenAIApiKey                string
		AzureOpenAIEndpoint              string
		AzureOpenAIGptDeploymentID       string
		AzureOpenAIEmbeddingDeploymentID string

		// aws bedrock
		AwsKey                      string
		AwsSecret                   string
		AwsBedrockModelArn          string
		AwsBedrockEmbeddingModelArn string

		Provider string

		Debug bool
	}

	GeneralChatCompletionMessage struct {
		Role    string
		Content string
	}
)

const (
	ProviderAzure   = "azure"
	ProviderOpenAI  = "openai"
	ProviderBedrock = "bedrock"
)

func (m GeneralChatCompletionMessage) Pretty() string {
	return fmt.Sprintf("{ Role: '%s', Content: '%s' }", m.Role, m.Content)
}

func New(cfg Config) *Instant {
	var openaiClient *openai.Client
	var azureOpenAIClient *azopenai.Client
	var bedrockClient bedrockruntimeiface.BedrockRuntimeAPI
	var err error

	if cfg.OpenAIApiKey != "" {
		openaiClient = openai.NewClient(cfg.OpenAIApiKey)
	}

	if cfg.AzureOpenAIApiKey != "" && cfg.AzureOpenAIEndpoint != "" && cfg.AzureOpenAIGptDeploymentID != "" {
		keyCredential := azcore.NewKeyCredential(cfg.AzureOpenAIApiKey)
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

func (s *Instant) RawRequest(ctx context.Context, messages []GeneralChatCompletionMessage) (string, error) {
	if s.cfg.Debug {
		slog.Info("[goutils.ai] RawRequest messages:")
		for _, message := range messages {
			slog.Info("[goutils.ai] RawRequest message", "message", message.Pretty())
		}
	}

	var ret string
	var err error

	switch s.cfg.Provider {
	case ProviderOpenAI:
		_messages := make([]openai.ChatCompletionMessage, 0, len(messages))
		for _, message := range messages {
			_messages = append(_messages, openai.ChatCompletionMessage{
				Role:    message.Role,
				Content: message.Content,
			})
		}
		ret, err = s.RawRequestOpenAI(ctx, _messages)

	case ProviderAzure:
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
		ret, err = s.RawRequestAzureOpenAI(ctx, _messages)

	case ProviderBedrock:
		_messages := make([]BedRockClaudeChatMessage, 0, len(messages))
		for _, message := range messages {
			_messages = append(_messages, BedRockClaudeChatMessage{
				Role: message.Role,
				Content: []BedRockClaudeMessageContent{
					{
						Type: "text",
						Text: message.Content,
					},
				},
			})
		}
		ret, err = s.RawRequestAWSBedrockClaude(ctx, _messages)

	default:
		return "", fmt.Errorf("provider %s not supported", s.cfg.Provider)
	}

	if err != nil {
		return "", err
	}
	if s.cfg.Debug {
		slog.Info("[goutils.ai] RawRequest", "ret", ret)
	}

	return ret, nil
}

func (s *Instant) OneTimeRequest(ctx context.Context, content string) (string, error) {
	return s.RawRequest(ctx, []GeneralChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: content,
		},
	})
}

func (s *Instant) OneTimeRequestJson(ctx context.Context, content string) (map[string]any, error) {
	text, err := s.RawRequest(ctx, []GeneralChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: content,
		},
	})
	if err != nil {
		return nil, err
	}
	return s.GrabJsonOutput(ctx, text)
}

func (s *Instant) TwoSteps(ctx context.Context, outputFormat string, input, inst string) (*ChainResult, error) {
	return s.MultipleSteps(ctx, ChainParams{
		Format: outputFormat,
		Steps: []ChainParamsStep{
			{
				Input:       input,
				Instruction: "",
			},
			{
				Input:       "",
				Instruction: inst,
			},
		},
	})
}

func (s *Instant) MultipleSteps(ctx context.Context, params ChainParams) (*ChainResult, error) {
	newSteps := make([]ChainParamsStep, 0)
	for _, step := range params.Steps {
		if step.Instruction == "" && step.Input == "" {
			continue
		}
		if step.Input != "" {
			inst := fmt.Sprintf("Please read the following text and just say the word \"OK\". Do not explain the text: \n\n  %s", step.Input)
			newSteps = append(newSteps, ChainParamsStep{
				Options:     nil,
				Input:       "",
				Instruction: inst,
			})
		} else if step.Instruction != "" {
			newSteps = append(newSteps, ChainParamsStep{
				Options:     nil,
				Input:       "",
				Instruction: step.Instruction,
			})
		}
	}
	params.Steps = newSteps
	return s.CallInChain(ctx, params)
}

func (s *Instant) CallInChain(ctx context.Context, params ChainParams) (*ChainResult, error) {
	ret := &ChainResult{}
	conv := make([]GeneralChatCompletionMessage, 0)
	for i := 0; i < len(params.Steps)-1; i++ {
		conv = append(conv, GeneralChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: params.Steps[i].Instruction,
		})

		resp, err := s.RawRequest(ctx, conv)
		if err != nil {
			return nil, err
		}
		conv = append(conv, GeneralChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: resp,
		})
	}

	finalStep := params.Steps[len(params.Steps)-1]
	conv = append(conv, GeneralChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: finalStep.Instruction,
	})

	resp, err := s.RawRequest(ctx, conv)
	if err != nil {
		return nil, err
	}

	if params.Format == "json" {
		js, err := s.GrabJsonOutput(ctx, resp)
		if err != nil {
			slog.Error("[goutils.ai] GrabJsonOutput error", "error", err)
			return nil, err
		}
		ret.Json = js
	}

	ret.Text = resp
	return ret, nil
}

func (s *Instant) GrabJsonOutput(ctx context.Context, input string, outputKeys ...string) (map[string]any, error) {
	// try to parse the response
	var resp map[string]any
	if err := json.Unmarshal([]byte(input), &resp); err != nil {
		slog.Warn("[goutils.ai] GrabJsonOutput error, let's try to extract the result", "input", input, "error", err)

		// use regex to extract the json part
		// it could be multiple lines
		re := regexp.MustCompile(`(?s)\{.*?\}`)
		input = re.FindString(input)
		// replace \\n -> \n
		input = regexp.MustCompile(`\\n`).ReplaceAllString(input, "\n")
		input = regexp.MustCompile(`\n`).ReplaceAllString(input, "")
		input = regexp.MustCompile(`\"`).ReplaceAllString(input, "\"")

		if err := json.Unmarshal([]byte(input), &resp); err != nil {
			slog.Error("[goutils.ai] GrabJsonOutput error again", "input", input, "error", err)
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
			return nil, nil
		}
		outputs[outputKey] = resp[outputKey]
	}

	return outputs, nil
}

func (s *Instant) GrabJsonOutputFromMd(ctx context.Context, input string, ptrOutput interface{}) error {
	if err := json.Unmarshal([]byte(input), ptrOutput); err != nil {
		slog.Warn("[goutils.ai] GrabJsonOutputRaw error, let's try to extract the result", "input", input, "error", err)

		input = strings.TrimSpace(input)

		if strings.Contains(input, "```json") {
			trimed, err := extractJSONFromMarkdown(input)
			if err != nil {
				slog.Warn("[goutils.ai] GrabJsonOutputFromMd error", "error", err)
			} else {
				input = trimed
			}
		}

		if err := json.Unmarshal([]byte(input), ptrOutput); err != nil {
			slog.Error("[goutils.ai] GrabJsonOutputFromMd error again", "input", input, "error", err)
			return err
		}
	}

	return nil
}

func (s *Instant) GetEmbeddings(ctx context.Context, input []string) ([]float32, error) {
	switch s.cfg.Provider {
	case ProviderAzure:
		vec, err := s.CreateEmbeddingAzureOpenAI(ctx, input)
		if err != nil {
			slog.Error("[goutils.ai] CreateEmbeddingAzureOpenAI error", "error", err)
			return nil, err
		}
		return vec, nil
	case ProviderOpenAI:
		return s.CreateEmbeddingOpenAI(ctx, input)
	case ProviderBedrock:
		return s.CreateEmbeddingBedrock(ctx, input)
	default:
		return nil, fmt.Errorf("provider %s not supported for embeddings", s.cfg.Provider)
	}
}

func extractJSONFromMarkdown(markdownContent string) (string, error) {
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	fmt.Printf("markdownContent: %v\n", markdownContent)

	// Parse the markdown content
	reader := text.NewReader([]byte(markdownContent))
	doc := md.Parser().Parse(reader)

	jsonContents := make([]string, 0)
	// Traverse the AST to find JSON code blocks
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Check if the node is a fenced code block
		if codeBlock, ok := n.(*ast.FencedCodeBlock); ok {
			// Get the language info
			lang := string(codeBlock.Language(reader.Source()))
			if lang == "json" {
				// Extract the content inside the code block
				content := codeBlock.Text(reader.Source())
				// Convert to string
				jsonContent := string(content)
				// Append to the list of JSON contents
				jsonContents = append(jsonContents, jsonContent)
				// Continue walking to find more JSON blocks
			}
		}

		return ast.WalkContinue, nil
	})

	if len(jsonContents) == 0 {
		slog.Error("[goutils.ai] No JSON code block found in the markdown content.")
	}
	return strings.Join(jsonContents, "\n"), nil
}
