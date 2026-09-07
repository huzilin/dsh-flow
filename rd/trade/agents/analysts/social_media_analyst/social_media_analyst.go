package social_media_analyst

import (
	"github.com/huzilin/trade/agents/analysts/sentiment_analyst"
	"github.com/huzilin/trade/llm"
)

// SocialMediaAnalyst 社交媒体分析师（向后兼容）
// 已重命名为 SentimentAnalyst，此类型保留用于向后兼容
type SocialMediaAnalyst = sentiment_analyst.SentimentAnalyst

// NewSocialMediaAnalyst 创建社交媒体分析师（已废弃，使用 sentiment_analyst.NewSentimentAnalyst）
func NewSocialMediaAnalyst(llmClient *llm.OpenAIClientV2, language string) *SocialMediaAnalyst {
	return sentiment_analyst.NewSentimentAnalyst(llmClient, language)
}
