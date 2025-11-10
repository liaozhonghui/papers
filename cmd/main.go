// 项目的主要入口
package main

import (
	"fmt"
	"os"
	"papers/internal/people"
	"strings"

	"github.com/spf13/cobra"
)

var (
	dateStr   string
	paperType string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "papers",
		Short: "人民日报系列PDF爬虫",
		Long:  `下载人民日报、健康时报、中国城市报、讽刺与幽默等报纸的PDF文件`,
		Run:   runCrawler,
	}

	// 添加命令行参数
	rootCmd.Flags().StringVarP(&dateStr, "date", "d", "", "指定日期，格式: YYYY-MM-DD (如: 2025-11-09)，不指定则使用今天")
	rootCmd.Flags().StringVarP(&paperType, "paper", "p", "", "报纸类型，多个用逗号分隔 (rmrb,jksb,zgcsb,fcyym)，不指定则下载所有")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCrawler(cmd *cobra.Command, args []string) {
	fmt.Println("=== 人民日报系列PDF爬虫 ===")

	// 解析报纸类型
	var paperTypes []string
	if paperType == "" {
		// 如果没有指定，下载所有类型
		paperTypes = []string{"rmrb", "jksb", "zgcsb"}
		fmt.Println("未指定报纸类型，将下载所有报纸")
	} else {
		// 按逗号分隔
		paperTypes = strings.Split(paperType, ",")
		// 去除空格
		for i, pt := range paperTypes {
			paperTypes[i] = strings.TrimSpace(pt)
		}
		fmt.Printf("指定报纸类型: %s\n", strings.Join(paperTypes, ", "))
	}

	// 显示日期信息
	if dateStr != "" {
		fmt.Printf("使用指定日期: %s\n", dateStr)
	} else {
		fmt.Println("未指定日期，使用今天的日期")
	}
	fmt.Println()

	// 记录成功和失败的数量
	successCount := 0
	failCount := 0

	// 遍历所有报纸类型
	for _, pt := range paperTypes {
		fmt.Printf("=== 开始爬取 %s ===\n", getPaperName(pt))

		// 创建爬虫实例
		crawler, err := people.NewCrawler(pt, dateStr)
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
			fmt.Printf("✓ %s 爬取完成!\n", getPaperName(pt))
			successCount++
		}
		fmt.Println()
	}

	// 显示总结
	fmt.Println("==================")
	fmt.Printf("任务完成! 成功: %d, 失败: %d\n", successCount, failCount)
}

// getPaperName 获取报纸的中文名称
func getPaperName(paperType string) string {
	names := map[string]string{
		"rmrb":  "人民日报",
		"jksb":  "健康时报",
		"zgcsb": "中国城市报",
		"fcyym": "讽刺与幽默",
	}
	if name, ok := names[paperType]; ok {
		return name
	}
	return paperType
}
