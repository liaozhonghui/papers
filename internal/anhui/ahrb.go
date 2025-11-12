package anhui

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// AHRBFetcher 安徽日报的特定获取逻辑
type AHRBFetcher struct {
	date time.Time
}

// NewAHRBFetcher 创建安徽日报获取器
func NewAHRBFetcher(date time.Time) *AHRBFetcher {
	return &AHRBFetcher{
		date: date,
	}
}

// BuildURL 构建指定版面的URL
// AHRB特点：没有pc路径，node_01.html格式（2位填充）
func (f *AHRBFetcher) BuildURL(page int) string {
	dateStr := f.date.Format("200601/02")
	return fmt.Sprintf("https://szb.ahnews.com.cn/ahrb/layout/%s/node_%02d.html", dateStr, page)
}

// GetPageCount 获取总版数
func (f *AHRBFetcher) GetPageCount(url string) (int, error) {
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
func (f *AHRBFetcher) FindPDFURL(doc *goquery.Document, baseURL string) (string, error) {
	var pdfURL string

	// 查找PDF链接 - 使用AHRB特定的选择器
	doc.Find("body > div.Newslistbox > div.Newsmain > div.newscon.clearfix > div.newsside > ul > li.oneclick1 > div > div > p:nth-child(1) > a:nth-child(2)").Each(func(i int, s *goquery.Selection) {
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
