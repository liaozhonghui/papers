package anhui

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// PCFetcher 江淮时报的特定获取逻辑
type PCFetcher struct {
	date time.Time
}

// NewPCFetcher 创建江淮时报获取器
func NewPCFetcher(date time.Time) *PCFetcher {
	return &PCFetcher{
		date: date,
	}
}

// BuildURL 构建指定版面的URL
// PC特点：有pc路径，node_1.html格式（无填充）
func (f *PCFetcher) BuildURL(page int) string {
	dateStr := f.date.Format("200601/02")
	return fmt.Sprintf("https://ahsbszb.ahnews.com.cn/pc/layout/%s/node_%d.html", dateStr, page)
}

// GetPageCount 获取总版数
func (f *PCFetcher) GetPageCount(url string) (int, error) {
	fmt.Println(url)
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
		// 默认设置为8版
		count = 8
	}

	return count, nil
}

// FindPDFURL 从页面中查找PDF下载链接
func (f *PCFetcher) FindPDFURL(doc *goquery.Document, baseURL string) (string, error) {
	var pdfURL string

	// 查找PDF链接 - PC特点：使用p标签的id="pdfUrl"
	pdfElement := doc.Find("#pdfUrl")
	if pdfElement.Length() > 0 {
		pdfURL = strings.TrimSpace(pdfElement.Text())
	}

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

	// 使用url.Parse解析相对路径
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("解析基础URL失败: %v", err)
	}

	pdfURLParsed, err := url.Parse(pdfURL)
	if err != nil {
		return "", fmt.Errorf("解析PDF URL失败: %v", err)
	}

	// 使用ResolveReference将相对路径转换为绝对路径
	absoluteURL := base.ResolveReference(pdfURLParsed)

	return absoluteURL.String(), nil
}
