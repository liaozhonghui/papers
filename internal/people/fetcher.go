package people

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Fetcher 人民日报系列报纸的获取逻辑实现
type Fetcher struct {
	paperType string // 报纸类型，如 rmrb(人民日报)、jksb(健康时报)、zgnyb(中国能源报)等
	baseURL   string
	date      time.Time
}

// NewFetcher 创建人民日报系列报纸的获取器
// paperType: 报纸类型，如 "rmrb"(人民日报)、"jksb"(健康时报)、"zgnyb"(中国能源报)等
func NewFetcher(paperType string, date time.Time) *Fetcher {
	return &Fetcher{
		paperType: paperType,
		baseURL:   fmt.Sprintf("https://paper.people.com.cn/%s/pc/layout", paperType),
		date:      date,
	}
}

// BuildURL 构建指定版面的URL
func (f *Fetcher) BuildURL(page int) string {
	dateStr := f.date.Format("200601/02")
	return fmt.Sprintf("%s/%s/node_%02d.html", f.baseURL, dateStr, page)
}

// GetPageCount 获取总版数
func (f *Fetcher) GetPageCount(url string) (int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return 0, err
	}

	// 查找版面数量
	count := 0
	doc.Find("body > div.main.w1000 > div.right.right-main > div.swiper-box > div > div").Each(func(i int, s *goquery.Selection) {
		count++
	})

	if count == 0 {
		// 尝试另一种方式：查找所有版面链接
		doc.Find(".swiper-slide").Each(func(i int, s *goquery.Selection) {
			count++
		})
	}

	if count == 0 {
		// 如果还是找不到，默认设置为8版（人民日报常见版数）
		count = 8
	}

	return count, nil
}

// FindPDFURL 从页面中查找PDF下载链接
func (f *Fetcher) FindPDFURL(doc *goquery.Document, baseURL string) (string, error) {
	var pdfURL string

	// 查找PDF链接
	doc.Find("body > div.main.w1000 > div.left.paper-box > div.paper-bot > p.right.btn > a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && strings.Contains(href, ".pdf") {
			pdfURL = href
		}
	})

	// 如果找不到，尝试其他选择器
	if pdfURL == "" {
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			text := s.Text()
			if exists && strings.Contains(href, ".pdf") && strings.Contains(text, "PDF") {
				pdfURL = href
			}
		})
	}

	if pdfURL == "" {
		return "", fmt.Errorf("未找到PDF链接")
	}

	// 如果是相对路径，使用url.Parse解析
	if !strings.HasPrefix(pdfURL, "http") {
		base, err := url.Parse(baseURL)
		if err != nil {
			return "", err
		}

		// 处理相对路径
		pdfPath, err := url.Parse(pdfURL)
		if err != nil {
			return "", err
		}

		// 解析相对URL
		fullURL := base.ResolveReference(pdfPath)
		pdfURL = fullURL.String()
	}

	return pdfURL, nil
}
