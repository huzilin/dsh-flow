package schemas

// PortfolioRating 5层评级，由研究经理和投资组合经理使用
type PortfolioRating string

const (
	RatingBuy       PortfolioRating = "Buy"
	RatingOverweight PortfolioRating = "Overweight"
	RatingHold      PortfolioRating = "Hold"
	RatingUnderweight PortfolioRating = "Underweight"
	RatingSell      PortfolioRating = "Sell"
)

// TraderAction 3层交易方向，由交易员使用
type TraderAction string

const (
	ActionBuy  TraderAction = "Buy"
	ActionHold TraderAction = "Hold"
	ActionSell TraderAction = "Sell"
)

// SentimentBand 情绪方向，由情绪分析师生成
type SentimentBand string

const (
	SentimentBullish       SentimentBand = "Bullish"
	SentimentMildlyBullish SentimentBand = "Mildly Bullish"
	SentimentNeutral       SentimentBand = "Neutral"
	SentimentMixed         SentimentBand = "Mixed"
	SentimentMildlyBearish SentimentBand = "Mildly Bearish"
	SentimentBearish       SentimentBand = "Bearish"
)

// Confidence 置信度级别
type Confidence string

const (
	ConfidenceLow    Confidence = "low"
	ConfidenceMedium Confidence = "medium"
	ConfidenceHigh   Confidence = "high"
)
