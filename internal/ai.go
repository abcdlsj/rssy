package internal

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

var (
	openaiClient *openai.Client

	OPENAI_API_KEY  = os.Getenv("OPENAI_API_KEY")
	OPENAI_ENDPOINT = os.Getenv("OPENAI_ENDPOINT")
)

func init() {
	cfg := openai.DefaultConfig(OPENAI_API_KEY)
	cfg.BaseURL = OPENAI_ENDPOINT
	openaiClient = openai.NewClientWithConfig(cfg)
}

func aiCompletion(prompt, content string) (string, error) {
	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: prompt},
				{Role: openai.ChatMessageRoleUser, Content: content},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to create completion: %v", err)
	}

	return resp.Choices[0].Message.Content, nil
}
