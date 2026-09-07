// Package dataflows 提供外部数据获取的路由层
//
// 数据流架构:
//
//	┌─────────────────────────────────────────────────────┐
//	│                   interface/router                 │
//	│              (统一路由层，RouteToVendor)             │
//	└─────────────────────────────────────────────────────┘
//	                        │
//	    ┌───────────────────┼───────────────────┐
//	    ▼                   ▼                   ▼
//	┌─────────┐        ┌─────────┐       ┌─────────┐
//	│ yfinance │        │   fred  │       │polymarket
//	└─────────┘        └─────────┘       └─────────┘
//
// 数据源支持:
//
//	| 数据类型       | 主要数据源      | Fallback       |
//	|--------------|---------------|----------------|
//	| OHLCV 行情   | Alpha Vantage | Yahoo Finance  |
//	| 技术指标     | Alpha Vantage | Yahoo Finance  |
//	| 基本面数据   | Alpha Vantage | Yahoo Finance  |
//	| 新闻         | Alpha Vantage | Yahoo Finance  |
//	| 宏观经济     | FRED          | -              |
//	| 预测市场     | Polymarket    | -              |
package dataflows
