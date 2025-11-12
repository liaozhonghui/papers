package anhui

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// XAWBFetcher 新安晚报的特定获取逻辑
type XAWBFetcher struct {
	date      time.Time
	outputDir string   // 用于存储临时JPG文件
	pageURLs  []string // 缓存所有版面的URL
}

// NewXAWBFetcher 创建新安晚报获取器
func NewXAWBFetcher(date time.Time, outputDir string) *XAWBFetcher {
	return &XAWBFetcher{
		date:      date,
		outputDir: outputDir,
		pageURLs:  make([]string, 0),
	}
}

// BuildURL 构建指定版面的URL
// XAWB特点：需要先获取首页，然后从版面列表中提取每个版面的URL
func (f *XAWBFetcher) BuildURL(page int) string {
	// 如果还没有获取版面URL列表，先获取
	if len(f.pageURLs) == 0 {
		urls, err := f.fetchPageURLs()
		if err != nil {
			// 如果获取失败，返回首页URL
			dateStr := f.date.Format("20060102")
			return fmt.Sprintf("http://epaper.ahwang.cn/xawb/%s/html/index.htm", dateStr)
		}
		f.pageURLs = urls
	}

	// 如果页码超出范围，返回首页
	if page < 1 || page > len(f.pageURLs) {
		dateStr := f.date.Format("20060102")
		return fmt.Sprintf("http://epaper.ahwang.cn/xawb/%s/html/index.htm", dateStr)
	}

	// 返回对应页码的URL（page从1开始）
	return f.pageURLs[page-1]
}

// fetchPageURLs 获取所有版面的URL列表
func (f *XAWBFetcher) fetchPageURLs() ([]string, error) {
	dateStr := f.date.Format("20060102")
	indexURL := fmt.Sprintf("http://epaper.ahwang.cn/xawb/%s/html/index.htm", dateStr)

	resp, err := http.Get(indexURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var pageURLs []string
	baseURL, _ := url.Parse(indexURL)

	// 从 #breakNewsList1 中提取所有版面链接
	doc.Find("#breakNewsList1 .bmml_con_div a.bmml_con_div_name").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && href != "" && href != "#" {
			// 解析相对路径
			pageURLParsed, err := url.Parse(href)
			if err == nil {
				absolutePageURL := baseURL.ResolveReference(pageURLParsed)
				pageURLs = append(pageURLs, absolutePageURL.String())
			}
		}
	})

	return pageURLs, nil
}

// GetPageCount 获取总版数
func (f *XAWBFetcher) GetPageCount(url string) (int, error) {
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

	// 查找版面数量 - 使用选择器 #breakNewsList1 下的版面列表
	count := 0
	doc.Find("#breakNewsList1 .bmml_con_div").Each(func(i int, s *goquery.Selection) {
		count++
	})

	if count == 0 {
		return 0, fmt.Errorf("未找到任何版面")
	}

	return count, nil
}

// FindPDFURL 从页面中查找PDF下载链接
// 对于XAWB，这个方法实际上是查找JPG图片URL，然后转换为PDF
func (f *XAWBFetcher) FindPDFURL(doc *goquery.Document, baseURL string) (string, error) {
	var imageURL string

	// 查找图片URL - 从 #sss > div 中提取 background-image
	doc.Find("#sss > div").Each(func(i int, s *goquery.Selection) {
		style, exists := s.Attr("style")
		if exists {
			// 使用正则提取 url("...") 中的URL
			re := regexp.MustCompile(`url\(["\']?([^"'\)]+)["\']?\)`)
			matches := re.FindStringSubmatch(style)
			if len(matches) > 1 {
				imageURL = matches[1]
			}
		}
	})

	// 如果没找到完整URL，尝试从img标签获取
	if imageURL == "" {
		doc.Find("#sss img").Each(func(i int, s *goquery.Selection) {
			src, exists := s.Attr("src")
			if exists && (strings.HasSuffix(src, ".jpg") || strings.HasSuffix(src, ".jpeg")) {
				imageURL = src
			}
		})
	}

	if imageURL == "" {
		return "", fmt.Errorf("未找到图片URL")
	}

	// 解析相对路径为绝对路径
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("解析基础URL失败: %v", err)
	}

	imgURLParsed, err := url.Parse(imageURL)
	if err != nil {
		return "", fmt.Errorf("解析图片URL失败: %v", err)
	}

	absoluteURL := base.ResolveReference(imgURLParsed)
	absoluteImageURL := absoluteURL.String()

	// 下载图片并转换为PDF
	pdfPath, err := f.downloadImageAndConvertToPDF(absoluteImageURL, baseURL)
	if err != nil {
		return "", fmt.Errorf("下载图片或转换PDF失败: %v", err)
	}

	// 返回本地PDF文件路径（使用特殊前缀标记为本地文件）
	return "file://" + pdfPath, nil
}

// downloadImageAndConvertToPDF 下载JPG图片并转换为PDF
func (f *XAWBFetcher) downloadImageAndConvertToPDF(imageURL, pageURL string) (string, error) {
	// 下载图片
	resp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载图片失败，HTTP状态码: %d", resp.StatusCode)
	}

	// 生成临时JPG文件名
	timestamp := time.Now().UnixNano()
	jpgPath := filepath.Join(f.outputDir, fmt.Sprintf("temp_%d.jpg", timestamp))

	// 保存JPG文件
	jpgFile, err := os.Create(jpgPath)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(jpgFile, resp.Body)
	jpgFile.Close()
	if err != nil {
		os.Remove(jpgPath)
		return "", err
	}

	// 转换为PDF
	pdfPath := filepath.Join(f.outputDir, fmt.Sprintf("temp_%d.pdf", timestamp))
	err = f.convertJPGToPDF(jpgPath, pdfPath)
	if err != nil {
		os.Remove(jpgPath)
		return "", err
	}

	// 删除临时JPG文件
	os.Remove(jpgPath)

	return pdfPath, nil
}

// convertJPGToPDF 将JPG图片转换为PDF文件
func (f *XAWBFetcher) convertJPGToPDF(jpgPath, pdfPath string) error {
	// 使用pdfcpu的ImportImages功能将图片转换为PDF
	conf := model.NewDefaultConfiguration()

	// ImportImages参数: sourceFiles(图片文件列表), outputFile, importConf, config
	err := api.ImportImagesFile([]string{jpgPath}, pdfPath, nil, conf)
	if err != nil {
		return fmt.Errorf("转换图片为PDF失败: %v", err)
	}

	return nil
}
