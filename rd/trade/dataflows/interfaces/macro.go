// Package interfaces 定义宏观经济数据请求结构
package interfaces

import "time"

// MacroRequest 宏观经济数据请求
type MacroRequest struct {
	Indicator    string             // 指标代码 (如 GDP, UNRATE)
	CurrentDate  time.Time          // 当前日期
	LookbackDays int                // 回溯天数
	Params       map[string]string  // 可选参数
}