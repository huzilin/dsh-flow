// Package models 定义新闻数据模型
package models

import "time"

// NewsItem 单条新闻
type NewsItem struct {
	Title       string
	Summary     string
	Source      string
	URL         string
	PublishDate time.Time
	Sentiment   *string // 可选：正面/负面/中性
}

// NewsResult 新闻结果
type NewsResult struct {
	Symbol string
	News   []*NewsItem
	Source string
}

// AnnouncementItem 公告项
type AnnouncementItem struct {
	Date  time.Time
	Title string
	Type  string // 公告类型
	URL   string
}

// AnnouncementResult 公告结果
type AnnouncementResult struct {
	Symbol        string
	Announcements []*AnnouncementItem
	Source        string
}

// FormatNewsTable 格式化新闻列表
func FormatNewsTable(news []*NewsItem, title string, limit int) string {
	if len(news) == 0 {
		return title + "\n\n无新闻数据"
	}

	output := title + "\n\n"
	output += "| Date | Title | Source |\n"
	output += "| --- | --- | --- |\n"

	count := min(len(news), limit)
	for i := 0; i < count; i++ {
		item := news[i]
		if item != nil {
			output += "| " + item.PublishDate.Format("2006-01-02") + " | " +
				item.Title + " | " +
				item.Source + "\n"
		}
	}

	if len(news) > limit {
		output += "\n*Showing " + formatIntInt(limit) + " of " + formatIntInt(len(news)) + " items*\n"
	}

	return output
}

// FormatAnnouncementTable 格式化公告列表
func FormatAnnouncementTable(announcements []*AnnouncementItem, title string, limit int) string {
	if len(announcements) == 0 {
		return title + "\n\n无公告数据"
	}

	output := title + "\n\n"
	output += "| Date | Title | Type |\n"
	output += "| --- | --- | --- |\n"

	count := min(len(announcements), limit)
	for i := 0; i < count; i++ {
		item := announcements[i]
		if item != nil {
			output += "| " + item.Date.Format("2006-01-02") + " | " +
				item.Title + " | " +
				item.Type + "\n"
		}
	}

	return output
}
