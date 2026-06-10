package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "{{ .ProjectName }}",
		Short: "{{ .ProjectName }} is a powerful CLI tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello from {{ .ProjectName }} CLI!")
		},
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
