package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/huzilin/trade/dataflows/interfaces"
	"github.com/huzilin/trade/dataflows/models"
	"github.com/huzilin/trade/dataflows/normalizer"
	"github.com/huzilin/trade/dataflows/vendors/tool_registry"
	"github.com/huzilin/trade/logger"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	PolymarketAPIBase      = "https://gamma-api.polymarket.com"
	PolymarketTimeout      = 30
	PolymarketDefaultLimit = 6
)

type PolymarketMarket struct {
	Question           string  `json:"question"`
	OutcomePrices      string  `json:"outcomePrices"`
	Outcomes           string  `json:"outcomes"`
	VolumeNum          float64 `json:"volumeNum"`
	EndDate            string  `json:"endDate"`
	Closed             bool    `json:"closed"`
	OneWeekPriceChange float64 `json:"oneWeekPriceChange"`
}

type PolymarketSearchResponse struct {
	Events []struct {
		Markets []PolymarketMarket `json:"markets"`
	} `json:"events"`
}

func parseJSONList(value string) []string {
	var result []string
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return []string{}
	}
	return result
}

func isForwardLooking(market PolymarketMarket, now time.Time) bool {
	if market.Closed {
		return false
	}

	if market.EndDate != "" {
		endTime, err := time.Parse(time.RFC3339, market.EndDate)
		if err == nil && endTime.Before(now) {
			return false
		}
	}

	prices := parseJSONList(market.OutcomePrices)
	outcomes := parseJSONList(market.Outcomes)

	return len(prices) > 0 && len(outcomes) > 0
}

func GetPredictionMarkets(ctx context.Context, topic string, limit int) (*models.PredictionResult, error) {
	url := fmt.Sprintf("%s/public-search?q=%s&limit_per_type=20", PolymarketAPIBase, topic)

	client := &http.Client{Timeout: time.Duration(PolymarketTimeout) * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.L().Warn("Polymarket search failed", zap.String("topic", topic), zap.Error(err))
		return nil, errors.Wrap(err, "polymarket request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data PolymarketSearchResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	now := time.Now()
	now = now.UTC()

	var items []*models.PredictionItem
	for _, event := range data.Events {
		for _, m := range event.Markets {
			if isForwardLooking(m, now) {
				prices := parseJSONList(m.OutcomePrices)

				yesPrice := decimal.NewFromFloat(0)
				noPrice := decimal.NewFromFloat(0)
				if len(prices) > 0 {
					prob, _ := strconv.ParseFloat(prices[0], 64)
					yesPrice = decimal.NewFromFloat(prob)
					if len(prices) > 1 {
						noProb, _ := strconv.ParseFloat(prices[1], 64)
						noPrice = decimal.NewFromFloat(noProb)
					} else {
						noPrice = decimal.NewFromFloat(1 - prob)
					}
				}

				endDate := m.EndDate
				if len(endDate) > 10 {
					endDate = endDate[:10]
				}
				endTime, _ := time.Parse("2006-01-02", endDate)

				items = append(items, &models.PredictionItem{
					Question:    m.Question,
					YesPrice:    yesPrice,
					NoPrice:     noPrice,
					Volume:      decimal.NewFromFloat(m.VolumeNum),
					EndDate:     endTime,
					Description: "",
				})
			}
		}
	}

	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Volume.GreaterThan(items[i].Volume) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	if limit > len(items) {
		limit = len(items)
	}

	return &models.PredictionResult{
		Topic:  topic,
		Items:  items[:limit],
		Source: "Polymarket",
	}, nil
}

func init() {
	tool_registry.Registry.RegisterGetPredictionMarkets("polymarket", GetPredictionMarkets)
}

type PolymarketProvider struct{}

func (p *PolymarketProvider) Name() string {
	return "polymarket"
}

func (p *PolymarketProvider) Capabilities() []interfaces.Capability {
	return []interfaces.Capability{
		interfaces.CapabilityPredictionMarkets,
	}
}

func (p *PolymarketProvider) Markets() []normalizer.MarketType {
	return []normalizer.MarketType{normalizer.MarketGlobal}
}

var _ interfaces.Provider = (*PolymarketProvider)(nil)
