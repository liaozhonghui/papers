package anhui

import (
	"papers/internal/crawler"
)

// NewCrawler 创建新的安徽日报系列爬虫实例
// paperType: 报纸类型
// fetcher: 特定报纸的获取逻辑实现
// dateStr: 可选的日期字符串，格式为 "2006-01-02" (如 "2025-11-10")
// 如果为空字符串，则使用当前东8区时间
func NewCrawler(paperType string, fetcher crawler.PaperFetcher, dateStr string) (*crawler.Crawler, error) {
	return crawler.NewCrawler(paperType, fetcher, dateStr)
}
