# Trading Agent Go

基于 Go 语言实现的交易数据获取与分析框架。

## 架构设计

### 核心模块

```
┌─────────────────────────────────────────────────────────────────┐
│                        工具层 (tools)                           │
│  对外暴露统一接口，调用路由层获取数据                              │
├─────────────────────────────────────────────────────────────────┤
│                        路由层 (router)                          │
│  根据 capability 和市场选择供应商并调用对应类型化函数                │
├─────────────────────────────────────────────────────────────────┤
│                    工具注册表 (tool_registry)                    │
│  管理所有供应商的类型化能力注册与查询                              │
├─────────────────────────────────────────────────────────────────┤
│                    供应商层 (vendors)                            │
│  akshare / yfinance / alpha_vantage / fred / eastmoney_mx / ...│
└─────────────────────────────────────────────────────────────────┘
```

## 工具注册表

### 设计理念

采用**类型化函数签名**替代传统的 `func(ctx context.Context, args ...any) (string, error)` 通用接口，实现编译时类型安全检查。

### 类型化能力定义

| 能力名称 | 函数类型 | 返回类型 |
|---------|---------|---------|
| GetStockData | `func(ctx, symbol, startDate, endDate string) ([]*models.Candle, error)` | `[]*models.Candle` |
| GetIndicators | `func(ctx, symbol, indicator, currDate string, lookBackDays int) (*models.IndicatorResult, error)` | `*models.IndicatorResult` |
| GetFundamentals | `func(ctx, symbol string, currDate time.Time) (*models.Fundamentals, error)` | `*models.Fundamentals` |
| GetBalanceSheet | `func(ctx, symbol, freq string, currDate time.Time) (*models.BalanceSheet, error)` | `*models.BalanceSheet` |
| GetCashflow | `func(ctx, symbol, freq string, currDate time.Time) (*models.Cashflow, error)` | `*models.Cashflow` |
| GetIncomeStatement | `func(ctx, symbol, freq string, currDate time.Time) (*models.IncomeStatement, error)` | `*models.IncomeStatement` |
| GetNews | `func(ctx, symbol string, startDate, endDate time.Time, limit int) (*models.NewsResult, error)` | `*models.NewsResult` |
| GetCompanyAnnouncements | `func(ctx, symbol string, startDate, endDate time.Time) (*models.AnnouncementResult, error)` | `*models.AnnouncementResult` |
| GetInsiderTransactions | `func(ctx, symbol string) (string, error)` | `string` |
| GetMacroIndicators | `func(ctx, indicator string, currDate time.Time, lookBackDays int, params map[string]string) (*models.MacroResult, error)` | `*models.MacroResult` |
| GetPredictionMarkets | `func(ctx, topic string, limit int) (*models.PredictionResult, error)` | `*models.PredictionResult` |
| GetVerifiedMarketSnapshot | `func(ctx, symbol string, currDate time.Time, lookBackDays int) (string, error)` | `string` |

### 能力合并原则

能力数量不应是"功能×市场"的笛卡尔积。同一功能在不同市场的实现应合并为单一能力，通过参数区分：

- **GetNews**：通过 `symbol` 参数区分个股新闻（symbol != ""）和市场/全球新闻（symbol == ""）

### 注册方式

每个供应商在 `init()` 函数中注册其能力：

```go
func init() {
    tool_registry.Registry.RegisterGetStockData("akshare", AkShareGetStockData)
    tool_registry.Registry.RegisterGetIndicators("akshare", AkShareGetIndicators)
    tool_registry.Registry.RegisterGetFundamentals("akshare", AkShareGetFundamentals)
    // ... 其他能力注册
}
```

### 查询方式

路由层通过注册表获取已注册的函数：

```go
fn, ok := tool_registry.Registry.GetGetStockData(vendor)
if !ok {
    return nil, errors.New("vendor not registered")
}
candles, err := fn(ctx, symbol, startDate, endDate)
```

## 供应商能力矩阵

| 供应商 | GetStockData | GetIndicators | GetFundamentals | GetBalanceSheet | GetCashflow | GetIncomeStatement | GetNews | GetCompanyAnnouncements | GetInsiderTransactions | GetMacroIndicators | GetPredictionMarkets | GetVerifiedMarketSnapshot |
|--------|-------------|--------------|-----------------|-----------------|-------------|--------------------|--------|-------------------------|------------------------|--------------------|----------------------|--------------------------|
| akshare | ✅ (A股) | ✅ (A股) | ✅ (A股) | ✅ (A股) | ✅ (A股) | ✅ (A股) | ✅ (A股) | ✅ (A股) | | | | |
| yfinance | ✅ (美股/港股/全球) | ✅ (美股/港股) | ✅ (美股/港股) | ✅ (美股/港股) | ✅ (美股/港股) | ✅ (美股/港股) | ✅ (美股/港股) | ✅ (美股/港股) | | | | ✅ |
| alpha_vantage | ✅ (美股) | | | | | | ✅ (全球) | | ✅ (美股) | | | |
| eastmoney_mx | | | | | | | ✅ (A股) | ✅ (A股) | | | | |
| fred | | | | | | | | | | ✅ | | |
| polymarket | | | | | | | | | | | ✅ | |

## 数据模型

### 核心数据结构

- `models.Candle` - K线数据
- `models.IndicatorResult` - 指标计算结果
- `models.Fundamentals` - 基本面数据
- `models.BalanceSheet` - 资产负债表
- `models.Cashflow` - 现金流量表
- `models.IncomeStatement` - 利润表
- `models.NewsResult` - 新闻结果
- `models.AnnouncementResult` - 公告结果
- `models.MacroResult` - 宏观数据
- `models.PredictionResult` - 预测市场数据

## 调用流程

### 获取股票数据

```go
// 1. 路由层选择供应商
vendor := router.SelectVendor("get_stock_data", "US") // 返回 "yfinance"

// 2. 获取类型化函数
fn, _ := tool_registry.Registry.GetGetStockData(vendor)

// 3. 调用类型化函数
candles, err := fn(ctx, "AAPL", "2024-01-01", "2024-01-31")
```

### 获取指标数据

```go
fn, _ := tool_registry.Registry.GetGetIndicators("akshare")
result, err := fn(ctx, "000001", "rsi", "2024-01-31", 90)
```

### 获取新闻

```go
// 获取个股新闻
fn, _ := tool_registry.Registry.GetGetNews("akshare")
result, err := fn(ctx, "000001", startDate, endDate, 10)

// 获取市场新闻（symbol 为空）
result, err := fn(ctx, "", startDate, endDate, 10)
```

## 代码规范

### 类型化函数命名

```go
// 供应商名 + 能力名
func AkShareGetStockData(ctx context.Context, symbol, startDate, endDate string) ([]*models.Candle, error)
func YFinanceGetStockData(ctx context.Context, symbol, startDate, endDate string) ([]*models.Candle, error)
```

### 注册函数命名

```go
// Register + 能力名
func (r *ToolRegistry) RegisterGetStockData(vendor string, fn GetStockDataFunc)
func (r *ToolRegistry) RegisterGetIndicators(vendor string, fn GetIndicatorsFunc)
```

### 查询函数命名

```go
// Get + 能力名
func (r *ToolRegistry) GetGetStockData(vendor string) (GetStockDataFunc, bool)
func (r *ToolRegistry) GetGetIndicators(vendor string) (GetIndicatorsFunc, bool)
```

### 内部函数命名

```go
// 内部函数不需要下划线前缀（Go 风格）
func postMX(url string, payload map[string]any) ([]byte, error)
func fetchStockNews(plainSymbol string) ([]map[string]any, error)
```

### 类型简化

```go
// 使用 any 替代 interface{}（Go 1.18+）
func marshalResult(result any) (string, error)
type Data map[string]any
```

## 错误处理

### 自定义错误类型

```go
// errs 包定义了供应商特定的错误类型
errs.NewNoMarketDataError(symbol, canonical, detail)
errs.NewVendorRateLimitError(vendor, symbol)
errs.NewVendorNotConfiguredError(vendor, message)
```

### 通用错误处理

```go
// 直接使用 github.com/pkg/errors 包
errors.Wrap(err, "failed to fetch data")
errors.Errorf("invalid parameter: %s", value)
errors.New("no data available")
```

## 模块目录

```
dataflows/
├── errs/                 # 错误定义（自定义错误类型）
├── indicators/           # 技术指标计算
├── interfaces/           # 接口定义（能力常量、请求/响应类型）
├── logger/               # 日志模块
├── market_symbols/       # 市场符号规范化
├── models/               # 数据模型
├── normalizer/           # 数据规范化
├── registry/             # 旧注册表（保留兼容）
├── router/               # 路由层
├── tools/                # 工具层接口
└── vendors/              # 供应商实现
    ├── akshare/          # AkShare 供应商（A股）
    ├── alpha_vantage/    # Alpha Vantage 供应商（美股/全球新闻）
    ├── eastmoney_mx/     # 东方财富MX供应商（A股新闻/公告）
    ├── fred/             # FRED 宏观数据供应商
    ├── polymarket/       # Polymarket 预测市场供应商
    ├── tool_registry/    # 类型化工具注册表
    ├── utils/            # 通用工具函数
    └── yfinance/         # Yahoo Finance 供应商（美股/港股/全球）
```

## 迁移指南

### 从旧接口迁移

旧接口：
```go
func AkShareGetStockData(ctx context.Context, args ...any) (string, error)
```

新接口：
```go
func AkShareGetStockData(ctx context.Context, symbol, startDate, endDate string) ([]*models.Candle, error)
```

### 注册迁移

旧注册：
```go
tool_registry.GlobalRegistry.Register("get_stock_data", "akshare", AkShareGetStockData)
```

新注册：
```go
tool_registry.Registry.RegisterGetStockData("akshare", AkShareGetStockData)
```

### 新闻能力迁移

旧接口（已删除）：
```go
func AkShareGetMarketNews(ctx context.Context, currDate time.Time, lookBackDays, limit int) (*models.NewsResult, error)
func AlphaVantageGlobalNews(ctx context.Context, currDate time.Time, lookBackDays, limit int) (*models.NewsResult, error)
```

新接口（统一为 GetNews）：
```go
func AkShareGetNews(ctx context.Context, symbol string, startDate, endDate time.Time, limit int) (*models.NewsResult, error)
func AlphaVantageGlobalNews(ctx context.Context, symbol string, startDate, endDate time.Time, limit int) (*models.NewsResult, error)
```
