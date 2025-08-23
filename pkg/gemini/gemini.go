package gemini

import (
	"context"
	"errors"
	"fmt"
	"time"

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

var (
	requestTimeout            = 10 * time.Second
	GeminiTooManyRequestError = errors.New("gemini api error: 429 Too Many Requests")
	GeminiEmptyAnswer         = errors.New("gemini api error: empty answer")
)

type Client struct {
	config  *config.Config
	ai      *genai.Client
	storage *storage.Storage
}

func NewClient(ctx context.Context, config *config.Config, storage *storage.Storage) (*Client, error) {
	ai, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  config.GeminiApiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create new gemini client: %w", err)
	}

	return &Client{
		config:  config,
		ai:      ai,
		storage: storage,
	}, nil
}

func (c *Client) GenerateContent(ctx context.Context, history storage.ConversationHistory, model, prompt string) (string, error) {
	requestContent := prepareRequest(history.Messages, prompt)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	resp, err := c.ai.Models.GenerateContent(ctxWithTimeout, model, requestContent, nil)
	if err != nil {
		if googleErr, ok := err.(*googleapi.Error); ok && googleErr.Code == 429 {
			return "", GeminiTooManyRequestError
		}

		return "", fmt.Errorf("gemini api error: %w", err)
	}

	var responseText string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				responseText += part.Text
			}
		}
	}

	if len(responseText) == 0 {
		return "", GeminiEmptyAnswer
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
			return nil, fmt.Errorf("cannot list models: %w", err)
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
