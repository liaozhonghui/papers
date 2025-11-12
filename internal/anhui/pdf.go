package anhui

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

// PaperFetcher 定义报纸特定的获取逻辑接口
type PaperFetcher interface {
	// BuildURL 构建指定版面的URL
	BuildURL(page int) string
	// GetPageCount 获取总版数
	GetPageCount(url string) (int, error)
	// FindPDFURL 从页面中查找PDF下载链接
	// baseURL: 当前页面的URL，用于解析相对路径
	FindPDFURL(doc *goquery.Document, baseURL string) (string, error)
}

// Crawler 日报爬虫基础结构
type Crawler struct {
	PaperType string // 报纸类型
	OutputDir string
	MergedDir string
	Date      time.Time
	PageCount int
	PDFFiles  []string
	Fetcher   PaperFetcher // 特定报纸的获取逻辑
}

// NewCrawler 创建新的爬虫实例
// fetcher: 特定报纸的获取逻辑实现
// dateStr: 可选的日期字符串，格式为 "2006-01-02" (如 "2025-11-10")
// 如果为空字符串，则使用当前东8区时间
func NewCrawler(paperType string, fetcher PaperFetcher, dateStr string) (*Crawler, error) {
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
		OutputDir: "web/files",
		MergedDir: mergedDir,
		Date:      targetDate,
		PDFFiles:  make([]string, 0),
		Fetcher:   fetcher,
	}, nil
}

// Run 执行爬虫任务
func (c *Crawler) Run() error {
	fmt.Printf("开始爬取%s PDF...\n", c.PaperType)

	// 创建输出目录
	if err := c.createDirectories(); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 获取版数
	url := c.Fetcher.BuildURL(1)
	pageCount, err := c.Fetcher.GetPageCount(url)
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

// downloadPDF 下载指定版面的PDF
func (c *Crawler) downloadPDF(page int) error {
	url := c.Fetcher.BuildURL(page)

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

	// 使用特定报纸的逻辑查找PDF链接，传入当前页面URL用于解析相对路径
	pdfURL, err := c.Fetcher.FindPDFURL(doc, url)
	if err != nil {
		return err
	}

	fmt.Printf("第 %d 版 PDF URL: %s\n", page, pdfURL)

	// 下载PDF文件
	return c.savePDF(pdfURL, page)
}

// savePDF 保存PDF文件
func (c *Crawler) savePDF(pdfURL string, page int) error {
	// 生成文件名: paperType_日期_版号.pdf
	filename := fmt.Sprintf("%s_%s_%02d.pdf", c.PaperType, c.Date.Format("20060102"), page)
	destPath := filepath.Join(c.OutputDir, filename)

	// 检查是否是本地文件（XAWB的情况）
	if strings.HasPrefix(pdfURL, "file://") {
		// 本地文件，直接移动或复制
		srcPath := strings.TrimPrefix(pdfURL, "file://")

		// 复制文件
		srcFile, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		if err != nil {
			return err
		}

		// 删除临时文件
		os.Remove(srcPath)

		c.PDFFiles = append(c.PDFFiles, destPath)
		return nil
	}

	// 网络URL，正常下载
	resp, err := http.Get(pdfURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP状态码: %d", resp.StatusCode)
	}

	// 创建文件
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// 写入内容
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	c.PDFFiles = append(c.PDFFiles, destPath)
	return nil
}

// mergePDFs 合并所有下载的PDF文件
func (c *Crawler) mergePDFs() error {
	if len(c.PDFFiles) == 0 {
		return fmt.Errorf("没有PDF文件需要合并")
	}

	// 输出文件名: paperType_日期.pdf
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
}

// GetDateString 获取日期字符串（用于测试）
func (c *Crawler) GetDateString() string {
	return c.Date.Format("2006-01-02")
}

// GetDate 获取日期对象
func (c *Crawler) GetDate() time.Time {
	return c.Date
}

// GetPaperType 获取报纸类型
func (c *Crawler) GetPaperType() string {
	return c.PaperType
}
