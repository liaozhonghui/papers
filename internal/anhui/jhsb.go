package anhui

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// JHSBFetcher 江淮时报的特定获取逻辑
type JHSBFetcher struct {
	date time.Time
}

// NewJHSBFetcher 创建江淮时报获取器
func NewJHSBFetcher(date time.Time) *JHSBFetcher {
	return &JHSBFetcher{
		date: date,
	}
}

// BuildURL 构建指定版面的URL
// JHSB特点：有pc路径，node_1.html格式（无填充）
func (f *JHSBFetcher) BuildURL(page int) string {
	dateStr := f.date.Format("200601/02")
	return fmt.Sprintf("https://szb.ahnews.com.cn/jhsb/pc/layout/%s/node_%d.html", dateStr, page)
}

// GetPageCount 获取总版数
func (f *JHSBFetcher) GetPageCount(url string) (int, error) {
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
func (f *JHSBFetcher) FindPDFURL(doc *goquery.Document, baseURL string) (string, error) {
	var pdfURL string

	// 查找PDF链接 - 使用JHSB特定的选择器
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
