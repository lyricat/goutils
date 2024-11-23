# goutils

some useful go utils

## Usage

```go
import "github.com/lyricat/goutils"
```

## packages

### Qdrant

```go
	qd := qdrant.New(qdrant.Config{
		Addr:	"Qdrant.Addr",
		APIKey: "Qdrant.APIKey",
	})
	if _, err := qd.Check(); err != nil {
		slog.Error("[index] qdrant check failed", "error", err)
	}
```

### AI

The `ai` package provides useful functions for AI related tasks, and it supports multiple AI providers:

- OpenAI
- OpenAI on Azure
- Claude AI on AWS Bedrock

```go
import "github.com/lyricat/goutils/ai"

	client := ai.New(ai.Config{
	AzureOpenAIApiKey:                "Azure.OpenAI.APIKey",
	AzureOpenAIEndpoint:              "Azure.OpenAI.Endpoint",
	AzureOpenAIGptDeploymentID:       "Azure.OpenAI.GptDeploymentID",
	AzureOpenAIEmbeddingDeploymentID: "Azure.OpenAI.EmbeddingDeploymentID",
	Provider:                         "azure",
	Debug:                            false,
})
```

#### One time API call for JSON response

Here is an example of how to use the `OneTimeRequest` function to check if a given content is spam:

```go
	inst := fmt.Sprintf(`Is it a spam ?:

	%s

	Output the probability as json, from 0 to 1.
	example:
	{ "probability": float_value, "is_spam": true_or_false}
	`, content)

	ret, err := client.OneTimeRequestJson(ctx, inst)
	if err != nil {
		slog.Error("failed to send one time request", "error", err)
	}

	spam, ok := ret["is_spam"].(bool)
	if !ok {
		slog.Error("no `is_spam` key in response")
	}
```

#### Multiple API calls for TEXT response

MultipleSteps function is used to send multiple requests to the AI provider, and the response of each request is used as the input of the next request.

Here is an example of how to use the `MultipleSteps` function to translate a given content to a specific language. In this case, we use `text` as the format of the response, because we want to get the translated text directly.

```go
	content := "The grapes are innocent, please do not ban the planting of grapes because eating grapes can lead to death."
	lang := "Japanese"
	// a timeout of 1 minutes context
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	inst := fmt.Sprintf(`You are an expert linguist, specializing in %s language.
Provide the %s translation for above text by ensuring:

1. accuracy (by correcting errors of addition, mistranslation, omission, or untranslated text),
2. fluency (by applying %s grammar, spelling and punctuation rules and ensuring there are no unnecessary repetitions),
3. style (by ensuring the translations always following the style of the original source text)
4. terminology (by ensuring terminology use is consistent and reflects the source text domain; and by only ensuring you use equivalent idioms of %s)
5. translation must be markdown format.

Do not provide any explanations or text apart from the translation result.
`, lang, lang, lang, lang)

	ret, err := client.MultipleSteps(ctx, ai.ChainParams{
		Format: "text",
		Steps: []ai.ChainParamsStep{
			{Input: content},
			{Instruction: inst},
		},
	})
	if err != nil {
		return "", err
	}

	slog.Info("translate result", "ret", ret.Text)
```

#### Get Text Embedding

> this function is only available for Azure OpenAI and OpenAI.

```go
	vector, err := a.aiInst.GetEmbeddings(ctx, []string{content})
	if err != nil {
		slog.Error("failed to get embeddings", "error", err)
	}
```

then you may use the vector to search at qdrant:

```go
import 	"github.com/lyricat/goutils/qdrant"

  // ...
	params := qdrant.SearchPointsParams{}
	params.CollectionName = QdrantCollectionName
	params.Vector = vector
	params.TopK = uint64(limit)
	searchResult, err := qd.SearchPointsWithFilter(ctx, params)
	if err != nil {
		slog.Error("failed to search points with filter", "error", err)
	}
```
