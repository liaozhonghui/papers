// 项目的主要入口
package main

import (
	"fmt"
	"os"
	"papers/internal/people"
)

func main() {
	fmt.Println("=== 人民日报PDF爬虫 ===")

	// 获取命令行参数中的日期（如果有）
	var dateStr string
	if len(os.Args) > 1 {
		dateStr = os.Args[1]
		fmt.Printf("使用指定日期: %s\n", dateStr)
	} else {
		fmt.Println("未指定日期，使用今天的日期")
	}

	// 创建爬虫实例
	crawler, err := people.NewCrawler(dateStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建爬虫失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "\n使用方法:\n")
		fmt.Fprintf(os.Stderr, "  %s [日期]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s              # 爬取今天的报纸\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s 2025-11-09   # 爬取指定日期的报纸\n", os.Args[0])
		os.Exit(1)
	}

	fmt.Printf("爬取日期: %s (东8区时间)\n", crawler.GetDateString())
	fmt.Println()

	// 执行爬虫任务
	if err := crawler.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n任务完成!")
}
