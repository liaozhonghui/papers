package people

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Crawler 人民日报爬虫
type Crawler struct {
	PaperType string // 报纸类型，如 rmrb(人民日报)、jksb(健康时报)、zgnyb(中国能源报)等
	BaseURL   string
	OutputDir string
	MergedDir string
	Date      time.Time
	PageCount int
	PDFFiles  []string
}

// NewCrawler 创建新的爬虫实例
// paperType: 报纸类型，如 "rmrb"(人民日报)、"jksb"(健康时报)、"zgnyb"(中国能源报)等
// dateStr: 可选的日期字符串，格式为 "2006-01-02" (如 "2025-11-10")
// 如果为空字符串，则使用当前东8区时间
func NewCrawler(paperType, dateStr string) (*Crawler, error) {
	var targetDate time.Time
	loc, _ := time.LoadLocation("Asia/Shanghai")

	if dateStr == "" {
		// 使用当前东8区时间
		targetDate = time.Now().In(loc)
	} else {
		// 解析传入的日期字符串
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("日期格式错误，应为 YYYY-MM-DD 格式: %v", err)
		}
		// 转换为东8区时间
		targetDate = parsedDate.In(loc)
	}

	// 创建日期目录路径
	dateDir := targetDate.Format("20060102")
	mergedDir := filepath.Join("dist", dateDir)

	return &Crawler{
		PaperType: paperType,
		BaseURL:   fmt.Sprintf("https://paper.people.com.cn/%s/pc/layout", paperType),
		OutputDir: "web/files",
		MergedDir: mergedDir,
		Date:      targetDate,
		PDFFiles:  make([]string, 0),
	}, nil
}

// Run 执行爬虫任务
func (c *Crawler) Run() error {
	fmt.Println("开始爬取人民日报PDF...")

	// 创建输出目录
	if err := c.createDirectories(); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 获取版数
	pageCount, err := c.getPageCount()
	if err != nil {
		return fmt.Errorf("获取版数失败: %v", err)
	}
	c.PageCount = pageCount
	fmt.Printf("共有 %d 版\n", pageCount)

	// 下载所有版面的PDF
	for i := 1; i <= pageCount; i++ {
		if err := c.downloadPDF(i); err != nil {
			fmt.Printf("下载第 %d 版失败: %v\n", i, err)
			continue
		}
		fmt.Printf("成功下载第 %d 版\n", i)
	}

	// 合并PDF
	if len(c.PDFFiles) > 0 {
		if err := c.mergePDFs(); err != nil {
			return fmt.Errorf("合并PDF失败: %v", err)
		}
		fmt.Println("PDF合并完成!")
	} else {
		return fmt.Errorf("没有下载到任何PDF文件")
	}

	return nil
}

// createDirectories 创建必要的目录
func (c *Crawler) createDirectories() error {
	dirs := []string{c.OutputDir, c.MergedDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// getPageCount 获取总版数
func (c *Crawler) getPageCount() (int, error) {
	url := c.buildURL(1)

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
	// 根据CSS选择器找到版面列表
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

// buildURL 构建指定版面的URL
func (c *Crawler) buildURL(page int) string {
	dateStr := c.Date.Format("200601/02")
	return fmt.Sprintf("%s/%s/node_%02d.html", c.BaseURL, dateStr, page)
}

// downloadPDF 下载指定版面的PDF
func (c *Crawler) downloadPDF(page int) error {
	url := c.buildURL(page)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	// 查找PDF链接
	var pdfURL string
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
		return fmt.Errorf("未找到PDF链接")
	}

	// 如果是相对路径，补全URL
	if !strings.HasPrefix(pdfURL, "http") {
		// 处理相对路径，如 ../../../attachement/...
		cleanPath := pdfURL
		for strings.HasPrefix(cleanPath, "../") {
			cleanPath = strings.TrimPrefix(cleanPath, "../")
		}

		// 若路径以 "attachement" 开头，补全前缀
		if strings.HasPrefix(cleanPath, "attachement") {
			pdfURL = fmt.Sprintf("https://paper.people.com.cn/%s/pc/%s", c.PaperType, cleanPath)
		} else {
			// 其它情况直接基于站点根路径补全
			pdfURL = "https://paper.people.com.cn/" + cleanPath
		}
	}

	fmt.Printf("第 %d 版 PDF URL: %s\n", page, pdfURL)

	// 下载PDF文件
	return c.savePDF(pdfURL, page)
}

// savePDF 保存PDF文件
func (c *Crawler) savePDF(url string, page int) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	// 生成文件名: paperType_日期_版号.pdf (如 rmrb_20250110_01.pdf)
	filename := fmt.Sprintf("%s_%s_%02d.pdf", c.PaperType, c.Date.Format("20060102"), page)
	filepath := filepath.Join(c.OutputDir, filename)

	// 创建文件
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// 写入内容
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	c.PDFFiles = append(c.PDFFiles, filepath)
	return nil
}

// mergePDFs 合并所有下载的PDF文件
func (c *Crawler) mergePDFs() error {
	if len(c.PDFFiles) == 0 {
		return fmt.Errorf("没有PDF文件需要合并")
	}

	// 输出文件名: paperType_日期.pdf (如 rmrb_20250110.pdf)
	outputFile := filepath.Join(c.MergedDir, fmt.Sprintf("%s_%s.pdf", c.PaperType, c.Date.Format("20060102")))

	// 如果输出文件已存在，先删除（确保可以覆盖）
	if _, err := os.Stat(outputFile); err == nil {
		fmt.Printf("删除已存在的合并文件: %s\n", outputFile)
		if err := os.Remove(outputFile); err != nil {
			return fmt.Errorf("删除已存在文件失败: %v", err)
		}
	}

	// 使用pdfcpu合并PDF
	conf := model.NewDefaultConfiguration()

	// 在合并前等待5秒
	fmt.Println("等待5秒后开始合并...")
	time.Sleep(5 * time.Second)

	// MergeCreateFile参数: inputFiles, outputFile, dividerPage(是否插入分隔页), config
	err := api.MergeCreateFile(c.PDFFiles, outputFile, false, conf)
	if err != nil {
		return err
	}

	fmt.Printf("合并后的文件保存至: %s\n", outputFile)

	fmt.Println("等待5秒后开始删除临时文件...")
	time.Sleep(5 * time.Second)

	// 合并成功后，删除所有临时PDF文件
	fmt.Println("清理临时文件...")
	for _, file := range c.PDFFiles {
		if err := os.Remove(file); err != nil {
			fmt.Printf("警告: 删除临时文件失败 %s: %v\n", file, err)
		}
	}
	fmt.Printf("已删除 %d 个临时PDF文件\n", len(c.PDFFiles))

	return nil
} // GetDateString 获取日期字符串（用于测试）
func (c *Crawler) GetDateString() string {
	return c.Date.Format("2006-01-02")
}
