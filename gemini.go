package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/api/googleapi"
)

// Chat struct holds the generative model and the chat session.
type Chat struct {
	model   *genai.GenerativeModel
	session *genai.ChatSession
}

// NewChat creates a new Chat instance.
func NewChat(ctx context.Context, modelName string) (*Chat, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	// Note: NewClient is deprecated, but NewGenerativeAIClient is not found.
	// Sticking with the one that works for now.
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel(modelName)
	session := model.StartChat()

	return &Chat{
		model:   model,
		session: session,
	}, nil
}

// GenerateContent sends a message to the model and returns the response.
func (c *Chat) GenerateContent(ctx context.Context, prompt string) (string, error) {
	resp, err := c.session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == 429 {
			return "", fmt.Errorf("gemini_api_error: 429 Too Many Requests")
		}
		return "", err
	}

	var responseText string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if txt, ok := part.(genai.Text); ok {
					responseText += string(txt)
				}
			}
		}
	}

	return responseText, nil
}

// ListModels returns a slice of available model names.
func ListModels(ctx context.Context) ([]string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var models []string
	iter := client.ListModels(ctx)
	for {
		m, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		models = append(models, m.Name)
	}
	return models, nil
}