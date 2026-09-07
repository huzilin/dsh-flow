// Package interfaces 定义预测市场数据请求结构
package interfaces

// PredictionRequest 预测市场请求
type PredictionRequest struct {
	Topic string // 查询主题
	Limit int    // 返回条数限制
}