package gemini

import (
	"context"
	"fmt"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/genai"

	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/config"
	"github.com/vasyvasilie/gemini-chat-tg-bot/pkg/storage"
)

const (
	ModelPrefix string = "models/"
	RoleUser    string = "user"
	RoleModel   string = "model"
)

type Client struct {
	cfg *config.Config
	ai  *genai.Client
}

func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	ai, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.GeminiApiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		cfg: cfg,
		ai:  ai,
	}, nil
}

func (c *Client) GenerateContent(ctx context.Context, history storage.ConversationHistory, model, prompt string) (string, error) {
	requestContent := prepareRequest(history.Messages, prompt)

	resp, err := c.ai.Models.GenerateContent(ctx, model, requestContent, nil)
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
				responseText += part.Text
			}
		}
	}

	return responseText, nil
}

func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	var models []string
	for m, err := range c.ai.Models.All(ctx) {
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

func prepareRequest(history []storage.Message, prompt string) []*genai.Content {
	content := []*genai.Content{}

	for _, msg := range history {
		role := RoleUser
		if msg.Role == RoleModel {
			role = RoleModel
		}

		content = append(content, &genai.Content{
			Parts: []*genai.Part{{Text: msg.Text}},
			Role:  role,
		})

	}

	content = append(content, &genai.Content{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  RoleUser,
	})

	return content
}
