package router

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/huzilin/trade/dataflows/config"
	"github.com/huzilin/trade/dataflows/errs"
	"github.com/huzilin/trade/dataflows/interfaces"
	"github.com/huzilin/trade/dataflows/models"
	"github.com/huzilin/trade/dataflows/normalizer"
	"github.com/huzilin/trade/dataflows/vendors/tool_registry"
	"github.com/huzilin/trade/logger"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type MethodCategory string

const (
	CategoryCoreStock   MethodCategory = "core_stock_apis"
	CategoryTechnical   MethodCategory = "technical_indicators"
	CategoryFundamental MethodCategory = "fundamental_data"
	CategoryNews        MethodCategory = "news_data"
	CategoryMacro       MethodCategory = "macro_data"
	CategoryPrediction  MethodCategory = "prediction_markets"
)

type Router struct {
	normalizer normalizer.SymbolNormalizer
	config     *config.Config
}

func NewRouter(norm normalizer.SymbolNormalizer, cfg *config.Config) *Router {
	return &Router{
		normalizer: norm,
		config:     cfg,
	}
}

func NewDefaultRouter() *Router {
	return NewRouter(
		normalizer.NewDefaultNormalizer(),
		config.Get(),
	)
}

func (r *Router) detectMarketVendor(symbol string) (normalizer.MarketType, string) {
	market := r.normalizer.DetectMarket(symbol)

	switch market {
	case normalizer.MarketAshare:
		return market, "akshare"
	case normalizer.MarketHKShare:
		return market, "yfinance"
	case normalizer.MarketUSShare:
		return market, "yfinance"
	default:
		return market, ""
	}
}

func (r *Router) getVendorForCapability(capability interfaces.Capability, market normalizer.MarketType, symbol string) []string {
	var vendors []string

	_, vendorName := r.detectMarketVendor(symbol)
	if vendorName != "" {
		vendors = append(vendors, vendorName)
	}

	switch capability {
	case interfaces.CapabilityStockData:
		switch market {
		case normalizer.MarketAshare:
			return append(vendors, "akshare")
		case normalizer.MarketHKShare:
			return append(vendors, "yfinance")
		case normalizer.MarketUSShare:
			return append(vendors, "yfinance", "alpha_vantage")
		}
	case interfaces.CapabilityIndicators:
		switch market {
		case normalizer.MarketAshare:
			return append(vendors, "akshare")
		case normalizer.MarketHKShare:
			return append(vendors, "yfinance")
		case normalizer.MarketUSShare:
			return append(vendors, "yfinance", "alpha_vantage")
		}
	case interfaces.CapabilityFundamentals:
		switch market {
		case normalizer.MarketAshare:
			return append(vendors, "akshare", "eastmoney_mx")
		case normalizer.MarketHKShare:
			return append(vendors, "yfinance")
		case normalizer.MarketUSShare:
			return append(vendors, "alpha_vantage", "yfinance")
		}
	case interfaces.CapabilityBalanceSheet:
		switch market {
		case normalizer.MarketAshare:
			return append(vendors, "akshare", "eastmoney_mx")
		case normalizer.MarketHKShare:
			return append(vendors, "yfinance")
		case normalizer.MarketUSShare:
			return append(vendors, "alpha_vantage", "yfinance")
		}
	case interfaces.CapabilityCashflow:
		switch market {
		case normalizer.MarketAshare:
			return append(vendors, "akshare", "eastmoney_mx")
		case normalizer.MarketHKShare:
			return append(vendors, "yfinance")
		case normalizer.MarketUSShare:
			return append(vendors, "alpha_vantage", "yfinance")
		}
	case interfaces.CapabilityIncomeStatement:
		switch market {
		case normalizer.MarketAshare:
			return append(vendors, "akshare", "eastmoney_mx")
		case normalizer.MarketHKShare:
			return append(vendors, "yfinance")
		case normalizer.MarketUSShare:
			return append(vendors, "alpha_vantage", "yfinance")
		}
	case interfaces.CapabilityNews:
		switch market {
		case normalizer.MarketAshare:
			return append(vendors, "eastmoney_mx", "akshare")
		case normalizer.MarketHKShare:
			return append(vendors, "yfinance")
		case normalizer.MarketUSShare:
			return append(vendors, "alpha_vantage", "yfinance")
		default:
			return append(vendors, "alpha_vantage")
		}
	case interfaces.CapabilityCompanyAnnouncements:
		switch market {
		case normalizer.MarketAshare:
			return append(vendors, "eastmoney_mx")
		case normalizer.MarketHKShare:
			return append(vendors, "yfinance")
		}
	case interfaces.CapabilityMacroIndicators:
		return append(vendors, "fred")
	case interfaces.CapabilityPredictionMarkets:
		return append(vendors, "polymarket")
	case interfaces.CapabilityVerifiedMarketSnapshot:
		return append(vendors, "yfinance")
	}

	return vendors
}

func (r *Router) GetStockData(ctx context.Context, symbol, startDate, endDate string) ([]*models.Candle, error) {
	market := r.normalizer.DetectMarket(symbol)
	vendors := r.getVendorForCapability(interfaces.CapabilityStockData, market, symbol)

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetStockData(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, symbol, startDate, endDate)
		if err != nil {
			if _, ok := err.(*errs.NoMarketDataError); ok {
				lastErr = err
				continue
			}
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if len(result) > 0 {
			return result, nil
		}
		lastErr = errs.NewNoMarketDataError(symbol, symbol, "no data points returned")
	}

	if _, ok := lastErr.(*errs.NoMarketDataError); ok {
		return nil, lastErr
	}

	return nil, errors.Errorf("no available vendor for get_stock_data, tried: %v", vendors)
}

func (r *Router) GetIndicators(ctx context.Context, symbol, indicator, currDate string, lookBackDays int) (*models.IndicatorResult, error) {
	market := r.normalizer.DetectMarket(symbol)
	vendors := r.getVendorForCapability(interfaces.CapabilityIndicators, market, symbol)

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetIndicators(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, symbol, indicator, currDate, lookBackDays)
		if err != nil {
			if _, ok := err.(*errs.NoMarketDataError); ok {
				lastErr = err
				continue
			}
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if result != nil {
			return result, nil
		}
		lastErr = errs.NewNoMarketDataError(symbol, symbol, "no indicator data")
	}

	if _, ok := lastErr.(*errs.NoMarketDataError); ok {
		return nil, lastErr
	}

	return nil, errors.Errorf("no available vendor for get_indicators, tried: %v", vendors)
}

func (r *Router) GetFundamentals(ctx context.Context, symbol string, currDate time.Time) (*models.Fundamentals, error) {
	market := r.normalizer.DetectMarket(symbol)
	vendors := r.getVendorForCapability(interfaces.CapabilityFundamentals, market, symbol)

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetFundamentals(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, symbol, currDate)
		if err != nil {
			if _, ok := err.(*errs.NoMarketDataError); ok {
				lastErr = err
				continue
			}
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if result != nil {
			return result, nil
		}
		lastErr = errs.NewNoMarketDataError(symbol, symbol, "no fundamentals data")
	}

	if _, ok := lastErr.(*errs.NoMarketDataError); ok {
		return nil, lastErr
	}

	return nil, errors.Errorf("no available vendor for get_fundamentals, tried: %v", vendors)
}

func (r *Router) GetBalanceSheet(ctx context.Context, symbol, freq string, currDate time.Time) (*models.BalanceSheet, error) {
	market := r.normalizer.DetectMarket(symbol)
	vendors := r.getVendorForCapability(interfaces.CapabilityBalanceSheet, market, symbol)

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetBalanceSheet(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, symbol, freq, currDate)
		if err != nil {
			if _, ok := err.(*errs.NoMarketDataError); ok {
				lastErr = err
				continue
			}
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if result != nil {
			return result, nil
		}
		lastErr = errs.NewNoMarketDataError(symbol, symbol, "no balance sheet data")
	}

	if _, ok := lastErr.(*errs.NoMarketDataError); ok {
		return nil, lastErr
	}

	return nil, errors.Errorf("no available vendor for get_balance_sheet, tried: %v", vendors)
}

func (r *Router) GetCashflow(ctx context.Context, symbol, freq string, currDate time.Time) (*models.Cashflow, error) {
	market := r.normalizer.DetectMarket(symbol)
	vendors := r.getVendorForCapability(interfaces.CapabilityCashflow, market, symbol)

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetCashflow(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, symbol, freq, currDate)
		if err != nil {
			if _, ok := err.(*errs.NoMarketDataError); ok {
				lastErr = err
				continue
			}
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if result != nil {
			return result, nil
		}
		lastErr = errs.NewNoMarketDataError(symbol, symbol, "no cashflow data")
	}

	if _, ok := lastErr.(*errs.NoMarketDataError); ok {
		return nil, lastErr
	}

	return nil, errors.Errorf("no available vendor for get_cashflow, tried: %v", vendors)
}

func (r *Router) GetIncomeStatement(ctx context.Context, symbol, freq string, currDate time.Time) (*models.IncomeStatement, error) {
	market := r.normalizer.DetectMarket(symbol)
	vendors := r.getVendorForCapability(interfaces.CapabilityIncomeStatement, market, symbol)

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetIncomeStatement(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, symbol, freq, currDate)
		if err != nil {
			if _, ok := err.(*errs.NoMarketDataError); ok {
				lastErr = err
				continue
			}
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if result != nil {
			return result, nil
		}
		lastErr = errs.NewNoMarketDataError(symbol, symbol, "no income statement data")
	}

	if _, ok := lastErr.(*errs.NoMarketDataError); ok {
		return nil, lastErr
	}

	return nil, errors.Errorf("no available vendor for get_income_statement, tried: %v", vendors)
}

func (r *Router) GetNews(ctx context.Context, symbol string, startDate, endDate time.Time, limit int) (*models.NewsResult, error) {
	market := r.normalizer.DetectMarket(symbol)
	vendors := r.getVendorForCapability(interfaces.CapabilityNews, market, symbol)

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetNews(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, symbol, startDate, endDate, limit)
		if err != nil {
			if _, ok := err.(*errs.NoMarketDataError); ok {
				lastErr = err
				continue
			}
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if result != nil {
			return result, nil
		}
		lastErr = errs.NewNoMarketDataError(symbol, symbol, "no news data")
	}

	if _, ok := lastErr.(*errs.NoMarketDataError); ok {
		return nil, lastErr
	}

	return nil, errors.Errorf("no available vendor for get_news, tried: %v", vendors)
}

func (r *Router) GetCompanyAnnouncements(ctx context.Context, symbol string, startDate, endDate time.Time) (*models.AnnouncementResult, error) {
	market := r.normalizer.DetectMarket(symbol)
	vendors := r.getVendorForCapability(interfaces.CapabilityCompanyAnnouncements, market, symbol)

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetCompanyAnnouncements(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, symbol, startDate, endDate)
		if err != nil {
			if _, ok := err.(*errs.NoMarketDataError); ok {
				lastErr = err
				continue
			}
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if result != nil {
			return result, nil
		}
		lastErr = errs.NewNoMarketDataError(symbol, symbol, "no company announcements")
	}

	if _, ok := lastErr.(*errs.NoMarketDataError); ok {
		return nil, lastErr
	}

	return nil, errors.Errorf("no available vendor for get_company_announcements, tried: %v", vendors)
}

func (r *Router) GetMacroIndicators(ctx context.Context, indicator string, currDate time.Time, lookBackDays int, params map[string]string) (*models.MacroResult, error) {
	vendors := r.getVendorForCapability(interfaces.CapabilityMacroIndicators, normalizer.MarketGlobal, "")

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetMacroIndicators(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, indicator, currDate, lookBackDays, params)
		if err != nil {
			if _, ok := err.(*errs.NoMarketDataError); ok {
				lastErr = err
				continue
			}
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if result != nil {
			return result, nil
		}
		lastErr = errs.NewNoMarketDataError(indicator, indicator, "no macro data")
	}

	if _, ok := lastErr.(*errs.NoMarketDataError); ok {
		return nil, lastErr
	}

	return nil, errors.Errorf("no available vendor for get_macro_indicators, tried: %v", vendors)
}

func (r *Router) GetPredictionMarkets(ctx context.Context, topic string, limit int) (*models.PredictionResult, error) {
	vendors := r.getVendorForCapability(interfaces.CapabilityPredictionMarkets, normalizer.MarketGlobal, "")

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetPredictionMarkets(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, topic, limit)
		if err != nil {
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if result != nil {
			return result, nil
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, errors.Errorf("no available vendor for get_prediction_markets, tried: %v", vendors)
}

func (r *Router) RouteWithNewRouter(ctx context.Context, method string, args ...any) (string, error) {
	capability := interfaces.Capability(method)

	switch capability {
	case interfaces.CapabilityStockData:
		if len(args) < 3 {
			return "", errors.New("get_stock_data requires 3 arguments: symbol, start_date, end_date")
		}
		symbol := fmt.Sprintf("%v", args[0])
		startDate := fmt.Sprintf("%v", args[1])
		endDate := fmt.Sprintf("%v", args[2])
		result, err := r.GetStockData(ctx, symbol, startDate, endDate)
		if err != nil {
			return "", err
		}
		jsonData, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(jsonData), nil

	case interfaces.CapabilityIndicators:
		if len(args) < 3 {
			return "", errors.New("get_indicators requires at least 3 arguments: symbol, indicator, curr_date")
		}
		symbol := fmt.Sprintf("%v", args[0])
		indicator := fmt.Sprintf("%v", args[1])
		currDate := fmt.Sprintf("%v", args[2])
		lookBackDays := 30
		if len(args) > 3 {
			lookBackDays = args[3].(int)
		}
		result, err := r.GetIndicators(ctx, symbol, indicator, currDate, lookBackDays)
		if err != nil {
			return "", err
		}
		jsonData, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(jsonData), nil

	case interfaces.CapabilityFundamentals:
		if len(args) < 1 {
			return "", errors.New("get_fundamentals requires 1 argument: symbol")
		}
		symbol := fmt.Sprintf("%v", args[0])
		currDate := time.Now()
		if len(args) > 1 {
			currDate = args[1].(time.Time)
		}
		result, err := r.GetFundamentals(ctx, symbol, currDate)
		if err != nil {
			return "", err
		}
		jsonData, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(jsonData), nil

	default:
		return "", errors.Errorf("capability %s not supported in RouteWithNewRouter", capability)
	}
}

func (r *Router) GetCapabilityMatrix() map[string][]string {
	result := make(map[string][]string)

	result[string(interfaces.CapabilityStockData)] = tool_registry.Registry.GetStockDataVendors()
	result[string(interfaces.CapabilityIndicators)] = tool_registry.Registry.GetIndicatorsVendors()
	result[string(interfaces.CapabilityFundamentals)] = tool_registry.Registry.GetFundamentalsVendors()
	result[string(interfaces.CapabilityBalanceSheet)] = tool_registry.Registry.GetBalanceSheetVendors()
	result[string(interfaces.CapabilityCashflow)] = tool_registry.Registry.GetCashflowVendors()
	result[string(interfaces.CapabilityIncomeStatement)] = tool_registry.Registry.GetIncomeStatementVendors()
	result[string(interfaces.CapabilityNews)] = tool_registry.Registry.GetNewsVendors()
	result[string(interfaces.CapabilityCompanyAnnouncements)] = tool_registry.Registry.GetCompanyAnnouncementsVendors()
	result[string(interfaces.CapabilityInsiderTransactions)] = tool_registry.Registry.GetInsiderTransactionsVendors()
	result[string(interfaces.CapabilityMacroIndicators)] = tool_registry.Registry.GetMacroIndicatorsVendors()
	result[string(interfaces.CapabilityPredictionMarkets)] = tool_registry.Registry.GetPredictionMarketsVendors()
	result[string(interfaces.CapabilityVerifiedMarketSnapshot)] = tool_registry.Registry.GetVerifiedMarketSnapshotVendors()

	return result
}

func (r *Router) GetInsiderTransactions(ctx context.Context, symbol string) (string, error) {
	market := r.normalizer.DetectMarket(symbol)
	vendors := r.getVendorForCapability(interfaces.CapabilityInsiderTransactions, market, symbol)

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetInsiderTransactions(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, symbol)
		if err != nil {
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if result != "" {
			return result, nil
		}
	}

	if lastErr != nil {
		return "", lastErr
	}

	return "", errors.Errorf("no available vendor for get_insider_transactions, tried: %v", vendors)
}

func (r *Router) GetVerifiedMarketSnapshot(ctx context.Context, symbol string, currDate time.Time, lookBackDays int) (string, error) {
	market := r.normalizer.DetectMarket(symbol)
	vendors := r.getVendorForCapability(interfaces.CapabilityVerifiedMarketSnapshot, market, symbol)

	var lastErr error
	for _, vendor := range vendors {
		fn, ok := tool_registry.Registry.GetGetVerifiedMarketSnapshot(vendor)
		if !ok {
			continue
		}

		result, err := fn(ctx, symbol, currDate, lookBackDays)
		if err != nil {
			logger.L().Warn("Vendor failed", zap.String("vendor", vendor), zap.Error(err))
			lastErr = err
			continue
		}

		if result != "" {
			return result, nil
		}
	}

	if lastErr != nil {
		return "", lastErr
	}

	return "", errors.Errorf("no available vendor for get_verified_market_snapshot, tried: %v", vendors)
}

func (r *Router) GetSupportedMethods() []string {
	return []string{
		string(interfaces.CapabilityStockData),
		string(interfaces.CapabilityIndicators),
		string(interfaces.CapabilityFundamentals),
		string(interfaces.CapabilityBalanceSheet),
		string(interfaces.CapabilityCashflow),
		string(interfaces.CapabilityIncomeStatement),
		string(interfaces.CapabilityNews),
		string(interfaces.CapabilityCompanyAnnouncements),
		string(interfaces.CapabilityInsiderTransactions),
		string(interfaces.CapabilityMacroIndicators),
		string(interfaces.CapabilityPredictionMarkets),
		string(interfaces.CapabilityVerifiedMarketSnapshot),
	}
}
