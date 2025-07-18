package internal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/sashabaranov/go-openai"
)

func getOpenAIClient() *openai.Client {
	adminPref, err := getAdminPreference()
	if err != nil || adminPref.OpenAIAPIKey == "" {
		return nil
	}

	cfg := openai.DefaultConfig(adminPref.OpenAIAPIKey)
	if adminPref.OpenAIEndpoint != "" {
		cfg.BaseURL = adminPref.OpenAIEndpoint
	}
	
	return openai.NewClientWithConfig(cfg)
}

func aiCompletion(prompt, content string) (string, error) {
	client := getOpenAIClient()
	if client == nil {
		return "", fmt.Errorf("OpenAI client not configured")
	}

	resp, err := client.CreateChatCompletion(
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
		return fmt.Errorf("no articles found for date %s", date.Format("2006-01-02"))
	}

	log.Infof("Generating summary for %d articles", len(articles))

	// 尝试使用AI，如果失败则使用简单总结
	var summary string
	var categories string
	var summaryType string
	
	client := getOpenAIClient()
	if client == nil {
		log.Infof("No OpenAI API key configured, using simple summary")
		summary = generateSimpleSummary(articles)
		categories = generateSimpleCategories(articles)
		summaryType = "Simple"
	} else {
		articlesText := formatArticlesForAI(articles)
		summary, err = aiCompletion(pref.AISummaryPrompt, articlesText)
		if err != nil {
			log.Errorf("AI completion failed, using simple summary instead: %v", err)
			summary = generateSimpleSummary(articles)
			categories = generateSimpleCategories(articles)
			summaryType = "Simple (AI failed)"
		} else {
			categories = extractCategories(summary)
			summaryType = "AI-generated"
		}
	}

	title := fmt.Sprintf("%s Summary - %s", summaryType, date.Format("2006-01-02"))
	
	err = createAISummary(email, date.Format("2006-01-02"), title, summary, categories, len(articles))
	if err != nil {
		return fmt.Errorf("failed to save AI summary: %v", err)
	}

	log.Infof("Generated %s summary for user %s on %s with %d articles", 
		summaryType, email, date.Format("2006-01-02"), len(articles))
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

func generateSimpleSummary(articles []Article) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("今日共有 %d 篇文章\n\n", len(articles)))
	
	// 按来源分组统计
	sourceCount := make(map[string]int)
	for _, article := range articles {
		sourceCount[article.Name]++
	}
	
	builder.WriteString("文章来源分布：\n")
	for source, count := range sourceCount {
		builder.WriteString(fmt.Sprintf("- %s: %d 篇\n", source, count))
	}
	
	builder.WriteString("\n主要文章：\n")
	for i, article := range articles {
		if i >= 5 { // 只显示前5篇
			break
		}
		builder.WriteString(fmt.Sprintf("%d. %s (来源: %s)\n", i+1, article.Title, article.Name))
	}
	
	if len(articles) > 5 {
		builder.WriteString(fmt.Sprintf("... 还有 %d 篇文章\n", len(articles)-5))
	}
	
	return builder.String()
}

func generateSimpleCategories(articles []Article) string {
	// 简单的分类逻辑：按来源分类
	sourceCount := make(map[string]int)
	for _, article := range articles {
		sourceCount[article.Name]++
	}
	
	var categories []string
	for source, count := range sourceCount {
		categories = append(categories, fmt.Sprintf("- %s: %d 篇文章", source, count))
	}
	
	return strings.Join(categories, "\n")
}
