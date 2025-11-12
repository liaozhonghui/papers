package papers

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "papers",
	Short: "中国报纸PDF爬虫工具",
	Long:  `一键下载并自动合并中国主流报纸的PDF版本`,
}

func init() {
	// 禁用自动生成的 completion 命令
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// 设置帮助命令的中文描述
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help [command]",
		Short: "查看命令帮助信息",
		Long:  `显示任何命令的详细帮助信息`,
	})
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
