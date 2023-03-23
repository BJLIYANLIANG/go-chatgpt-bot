package main

import (
	"context"
	"fmt"
	"github.com/penwyp/go-chatgpt-bot/core"
	"github.com/spf13/cobra"
)

var rootCommand = &cobra.Command{
	Use:   "chatgpt-bot",
	Short: "ChatGPT 机器人",
	Long:  fmt.Sprintf(""),
}

// main 执行真正业务逻辑
func main() {
	addCommand(rootCommand, core.ChatGPTCommand)
	cobra.CheckErr(rootCommand.ExecuteContext(context.Background()))
}

func addCommand(root *cobra.Command, cmd *cobra.Command) {
	root.AddCommand(cmd)
}
