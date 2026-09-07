package fred

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/huzilin/trade/dataflows/config"
	"github.com/huzilin/trade/dataflows/errs"
	"github.com/huzilin/trade/dataflows/interfaces"
	"github.com/huzilin/trade/dataflows/models"
	"github.com/huzilin/trade/dataflows/normalizer"
	"github.com/huzilin/trade/dataflows/vendors/tool_registry"
	"github.com/huzilin/trade/dataflows/vendors/utils"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	FREDAPIBase         = "https://api.stlouisfed.org/fred"
	FREDRequestTimeout  = 30
	FREDDefaultLookback = 365
	FREDMaxRows         = 40
)

var fredSeriesAliases = map[string]string{
	"fed_funds_rate":         "FEDFUNDS",
	"federal_funds_rate":     "FEDFUNDS",
	"fed_funds":              "FEDFUNDS",
	"2y_treasury":            "DGS2",
	"10y_treasury":           "DGS10",
	"30y_treasury":           "DGS30",
	"10y_2y_spread":          "T10Y2Y",
	"yield_curve":            "T10Y2Y",
	"cpi":                    "CPIAUCSL",
	"core_cpi":               "CPILFESL",
	"pce":                    "PCEPI",
	"core_pce":               "PCEPILFE",
	"inflation_expectations": "T10YIE",
	"real_gdp":               "GDPC1",
	"gdp":                    "GDP",
	"industrial_production":  "INDPRO",
	"unemployment_rate":      "UNRATE",
	"unemployment":           "UNRATE",
	"nonfarm_payrolls":       "PAYEMS",
	"payrolls":               "PAYEMS",
	"initial_claims":         "ICSA",
	"m2":                     "M2SL",
	"money_supply":           "M2SL",
	"vix":                    "VIXCLS",
	"dollar_index":           "DTWEXBGS",
	"consumer_sentiment":     "UMCSENT",
	"housing_starts":         "HOUST",
	"retail_sales":           "RSAFS",
}

func FredAPIKey() (string, error) {
	cfg := config.Get()
	if cfg.FREDAPIKey == "" {
		return "", errs.NewVendorNotConfiguredError("fred",
			"FRED_API_KEY environment variable is not set. Get a free key at https://fred.stlouisfed.org/docs/api/api_key.html.")
	}
	return cfg.FREDAPIKey, nil
}

func ResolveSeriesID(indicator string) string {
	key := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(indicator, " ", "_"), "-", "_"))
	if seriesID, ok := fredSeriesAliases[key]; ok {
		return seriesID
	}
	return strings.ToUpper(indicator)
}

func FredRequest(path string, params map[string]string) ([]byte, error) {
	apiKey, err := FredAPIKey()
	if err != nil {
		return nil, err
	}

	params["api_key"] = apiKey
	params["file_type"] = "json"

	baseURL, _ := url.Parse(FREDAPIBase + "/" + path)
	query := baseURL.Query()
	for k, v := range params {
		query.Set(k, v)
	}
	baseURL.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(FREDRequestTimeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create FRED request")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "FRED request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("FRED request failed: %s", string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read FRED response")
	}
	return data, nil
}

type FredSeriesInfo struct {
	Title       string `json:"title"`
	Units       string `json:"units_short"`
	Frequency   string `json:"frequency"`
	SeasonalAdj string `json:"seasonal_adjustment_short"`
}

type FredObservation struct {
	Date  string `json:"date"`
	Value string `json:"value"`
}

func GetFredSeriesInfo(seriesID string) (*FredSeriesInfo, error) {
	params := map[string]string{"series_id": seriesID}
	data, err := FredRequest("series", params)
	if err != nil {
		return nil, errors.Wrap(err, "FRED series info request failed")
	}

	var result struct {
		Seriess []FredSeriesInfo `json:"seriess"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse FRED series info")
	}

	if len(result.Seriess) == 0 {
		return nil, errors.Errorf("FRED series %q not found", seriesID)
	}

	return &result.Seriess[0], nil
}

func GetFredObservations(seriesID, startDate, endDate string) ([]FredObservation, error) {
	params := map[string]string{
		"series_id":         seriesID,
		"observation_start": startDate,
		"observation_end":   endDate,
		"sort_order":        "asc",
	}
	data, err := FredRequest("series/observations", params)
	if err != nil {
		return nil, errors.Wrap(err, "FRED observations request failed")
	}

	var result struct {
		Observations []FredObservation `json:"observations"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, errors.Wrap(err, "failed to parse FRED observations")
	}

	observations := make([]FredObservation, 0)
	for _, o := range result.Observations {
		if o.Value != "." && o.Value != "" {
			observations = append(observations, o)
		}
	}

	return observations, nil
}

func GetMacroData(ctx context.Context, indicator string, currDate time.Time, lookBackDays int, params map[string]string) (*models.MacroResult, error) {
	startDate := utils.FormatDate(utils.AddDays(currDate, -lookBackDays))
	seriesID := ResolveSeriesID(indicator)

	info, err := GetFredSeriesInfo(seriesID)
	if err != nil {
		return nil, errors.Wrap(err, "FRED series info request failed")
	}

	observations, err := GetFredObservations(seriesID, startDate, utils.FormatDate(currDate))
	if err != nil {
		return nil, errors.Wrap(err, "FRED observations request failed")
	}

	if len(observations) == 0 {
		return nil, errs.NewNoMarketDataError(indicator, seriesID, "no observations")
	}

	data := make([]*models.MacroIndicator, 0, len(observations))
	for _, o := range observations {
		val, _ := decimal.NewFromString(o.Value)
		date, _ := time.Parse("2006-01-02", o.Date)
		data = append(data, &models.MacroIndicator{
			Date:  date,
			Value: val,
		})
	}

	return &models.MacroResult{
		Indicator: indicator,
		Values:    data,
		Source:    "FRED",
		Unit:      info.Units,
	}, nil
}

func init() {
	tool_registry.Registry.RegisterGetMacroIndicators("fred", GetMacroData)
}

type FredProvider struct{}

func (p *FredProvider) Name() string {
	return "fred"
}

func (p *FredProvider) Capabilities() []interfaces.Capability {
	return []interfaces.Capability{
		interfaces.CapabilityMacroIndicators,
	}
}

func (p *FredProvider) Markets() []normalizer.MarketType {
	return []normalizer.MarketType{normalizer.MarketGlobal}
}

var _ interfaces.Provider = (*FredProvider)(nil)
