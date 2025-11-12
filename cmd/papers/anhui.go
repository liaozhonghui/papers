package papers

import (
	"fmt"
	"os"
	"papers/internal/anhui"
	"strings"

	"github.com/spf13/cobra"
)

var anhuiCmd = &cobra.Command{
	Use:   "anhui",
	Short: "安徽日报系列PDF爬虫",
	Long:  `下载安徽日报、安徽商报等报纸的PDF文件`,
	Run:   runanhuiCrawler,
}

var (
	anhuiDateStr   string
	anhuiPaperType string
)

func init() {
	// 添加命令行参数
	anhuiCmd.Flags().StringVarP(&anhuiDateStr, "date", "d", "", "指定日期，格式: YYYY-MM-DD (如: 2025-11-09)，不指定则使用今天")
	anhuiCmd.Flags().StringVarP(&anhuiPaperType, "paper", "p", "", "报纸类型，多个用逗号分隔 (rmrb,jksb,zgcsb,fcyym)，不指定则下载所有")

	rootCmd.AddCommand(anhuiCmd)
}

func runanhuiCrawler(cmd *cobra.Command, args []string) {
	fmt.Println("=== 安徽日报系列PDF爬虫 ===")

	// 解析报纸类型
	var anhuiPaperTypes []string
	if anhuiPaperType == "" {
		// 如果没有指定，下载所有类型
		anhuiPaperTypes = []string{"ahrb", "ncb", "jhsb", "fzb", "pc"}
		fmt.Println("未指定报纸类型，将下载所有报纸")
	} else {
		// 按逗号分隔
		anhuiPaperTypes = strings.Split(anhuiPaperType, ",")
		// 去除空格
		for i, pt := range anhuiPaperTypes {
			anhuiPaperTypes[i] = strings.TrimSpace(pt)
		}
		fmt.Printf("指定报纸类型: %s\n", strings.Join(anhuiPaperTypes, ", "))
	}

	// 显示日期信息
	if anhuiDateStr != "" {
		fmt.Printf("使用指定日期: %s\n", anhuiDateStr)
	} else {
		fmt.Println("未指定日期，使用今天的日期")
	}
	fmt.Println()

	// 记录成功和失败的数量
	successCount := 0
	failCount := 0

	// 遍历所有报纸类型
	for _, pt := range anhuiPaperTypes {
		fmt.Printf("=== 开始爬取 %s ===\n", getAnhuiPaperName(pt))

		// 创建对应的Fetcher
		var fetcher anhui.PaperFetcher
		var crawler *anhui.Crawler
		var err error

		// 先创建爬虫实例获取日期
		tempCrawler, err := anhui.NewCrawler(pt, nil, anhuiDateStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "创建爬虫失败 (%s): %v\n", pt, err)
			failCount++
			fmt.Println()
			continue
		}

		// 根据报纸类型创建对应的Fetcher
		switch pt {
		case "ahrb":
			fetcher = anhui.NewAHRBFetcher(tempCrawler.GetDate())
		case "ncb":
			fetcher = anhui.NewNCBFetcher(tempCrawler.GetDate())
		case "jhsb":
			fetcher = anhui.NewJHSBFetcher(tempCrawler.GetDate())
		case "fzb":
			fetcher = anhui.NewFZBFetcher(tempCrawler.GetDate())
		case "pc":
			fetcher = anhui.NewPCFetcher(tempCrawler.GetDate())
		default:
			fmt.Fprintf(os.Stderr, "未知的报纸类型: %s\n", pt)
			failCount++
			fmt.Println()
			continue
		}

		// 重新创建带Fetcher的爬虫实例
		crawler, err = anhui.NewCrawler(pt, fetcher, anhuiDateStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "创建爬虫失败 (%s): %v\n", pt, err)
			failCount++
			fmt.Println()
			continue
		}

		fmt.Printf("爬取日期: %s (东8区时间)\n", crawler.GetDateString())

		// 执行爬虫任务
		if err := crawler.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "爬取失败 (%s): %v\n", pt, err)
			failCount++
		} else {
			fmt.Printf("✓ %s 爬取完成!\n", getAnhuiPaperName(pt))
			successCount++
		}
		fmt.Println()
	}

	// 显示总结
	fmt.Println("==================")
	fmt.Printf("任务完成! 成功: %d, 失败: %d\n", successCount, failCount)
}

// getAnhuiPaperName 获取报纸的中文名称
func getAnhuiPaperName(anhuiPaperType string) string {
	names := map[string]string{
		"ahrb": "安徽日报",
		"ncb":  "安徽日报农村版",
		"jhsb": "江淮时报",
		"fzb":  "安徽法治报",
		"pc":   "安徽商报",
	}
	if name, ok := names[anhuiPaperType]; ok {
		return name
	}
	return anhuiPaperType
}
