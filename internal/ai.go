package internal

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
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

func generateDailyAISummary(email string, date time.Time) error {
	if OPENAI_API_KEY == "" {
		return fmt.Errorf("OPENAI_API_KEY is not set")
	}

	pref, err := getUserPreference(email)
	if err != nil {
		return fmt.Errorf("failed to get user preference: %v", err)
	}

	if !pref.EnableAISummary {
		return fmt.Errorf("AI summary is disabled for user %s", email)
	}

	articles, err := getArticlesForAISummary(email, date)
	if err != nil {
		return fmt.Errorf("failed to get articles: %v", err)
	}

	if len(articles) == 0 {
		log.Infof("No articles found for AI summary for user %s on %s", email, date.Format("2006-01-02"))
		return nil
	}

	articlesText := formatArticlesForAI(articles)
	summary, err := aiCompletion(pref.AISummaryPrompt, articlesText)
	if err != nil {
		return fmt.Errorf("failed to generate AI summary: %v", err)
	}

	title := fmt.Sprintf("Daily Summary - %s", date.Format("2006-01-02"))
	categories := extractCategories(summary)

	err = createAISummary(email, date.Format("2006-01-02"), title, summary, categories, len(articles))
	if err != nil {
		return fmt.Errorf("failed to save AI summary: %v", err)
	}

	log.Infof("Generated AI summary for user %s on %s with %d articles", email, date.Format("2006-01-02"), len(articles))
	return nil
}

func formatArticlesForAI(articles []Article) string {
	var builder strings.Builder
	builder.WriteString("以下是今天的RSS文章列表：\n\n")

	for i, article := range articles {
		builder.WriteString(fmt.Sprintf("%d. 标题：%s\n", i+1, article.Title))
		builder.WriteString(fmt.Sprintf("   来源：%s\n", article.Name))
		builder.WriteString(fmt.Sprintf("   链接：%s\n", article.Link))
		if article.Content != "" && len(article.Content) > 100 {
			content := article.Content
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			builder.WriteString(fmt.Sprintf("   内容摘要：%s\n", content))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func extractCategories(summary string) string {
	lines := strings.Split(summary, "\n")
	var categories []string
	inCategorySection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "分类") || strings.Contains(line, "类别") || strings.Contains(line, "Categories") {
			inCategorySection = true
			continue
		}

		if inCategorySection && line != "" {
			if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") || strings.HasPrefix(line, "*") {
				categories = append(categories, line)
			} else if strings.Contains(line, "：") || strings.Contains(line, ":") {
				categories = append(categories, line)
			} else if len(categories) > 0 {
				break
			}
		}
	}

	return strings.Join(categories, "\n")
}
