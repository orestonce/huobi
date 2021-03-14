package main

import (
	"github.com/spf13/cobra"
	"github.com/orestonce/huobi"
	"log"
)

var rootCmd = &cobra.Command{
	Use: "huobi",
}

func init() {
	rootCmd.AddCommand(huobi.InstallCmd)
	rootCmd.AddCommand(huobi.CollectMessageCmd)
	rootCmd.AddCommand(huobi.SearchCmd)
	rootCmd.AddCommand(huobi.WatchCmd)
	rootCmd.Run = rootCmd.HelpFunc()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Println("error", err)
	}
}
