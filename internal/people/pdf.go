package people

import (
	"papers/internal/crawler"
)

// NewCrawler 创建新的人民日报系列爬虫实例
// paperType: 报纸类型，如 "rmrb"(人民日报)、"jksb"(健康时报)、"zgnyb"(中国能源报)等
// dateStr: 可选的日期字符串，格式为 "2006-01-02" (如 "2025-11-10")
// 如果为空字符串，则使用当前东8区时间
func NewCrawler(paperType, dateStr string) (*crawler.Crawler, error) {
	// 创建爬虫实例以获取日期
	tempCrawler, err := crawler.NewCrawler(paperType, nil, dateStr)
	if err != nil {
		return nil, err
	}

	// 创建人民日报系列的Fetcher
	fetcher := NewFetcher(paperType, tempCrawler.GetDate())

	// 使用正确的Fetcher重新创建爬虫
	return crawler.NewCrawler(paperType, fetcher, dateStr)
}
