# Google Custom Search Go Module

This Go module provides a client for interacting with the [Google Custom Search JSON API](https://developers.google.com/custom-search/v1/overview). It allows performing keyword-based searches with pagination support.

## Features

- Search Google using the Custom Search JSON API
- Supports pagination
- Uses `log/slog` for structured logging
- Handles errors properly with timeout enforcement (30 seconds)

## Installation

To use this module, you need to install it in your Go project:

```sh
go get github.com/yourusername/googlesearch
```

## Usage

### Prerequisites

1. Obtain a Google API Key from the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a Custom Search Engine and retrieve its `cx` identifier

### Example

```go
package main

import (
	"fmt"
	"log/slog"
	"github.com/yourusername/googlesearch"
)

func main() {
	apiKey := "your-api-key"
	cx := "your-search-engine-id"
	client := googlesearch.NewSearchClient(apiKey, cx)

	query := "who is the president of the united states"
	start := 1

	result, err := client.Search(query, start)
	if err != nil {
		slog.Error("Search failed", "error", err)
		return
	}

	for _, item := range result.Items {
		fmt.Printf("Title: %s\nLink: %s\nSnippet: %s\n\n", item.Title, item.Link, item.Snippet)
	}
}
```

## API Reference

### `NewSearchClient(apiKey, cx string) *SearchClient`

Creates a new search client with the given API key and search engine ID.

### `Search(query string, start int) (*SearchResponse, error)`

Executes a search query with pagination.

#### Parameters:

- `query`: The search keyword
- `start`: The result index for pagination (e.g., 1, 11, 21 for pages 1, 2, 3...)

#### Returns:

- `SearchResponse`: A struct containing search results
- `error`: Any error encountered during the request

## Error Handling

- Logs errors using `log/slog`
- Returns meaningful error messages for HTTP failures

