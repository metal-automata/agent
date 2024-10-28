package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	cobradoc "github.com/spf13/cobra/doc"
)

var cmdGenerateDocs = &cobra.Command{
	Use:   "gendocs",
	Short: "Generate markdown docs for Agent",
	Run: func(cmd *cobra.Command, _ []string) {
		err := os.MkdirAll("docs", os.ModeSticky|os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		linkHandler := func(s string) string { return s }
		// nolint:revive // method is passed as a parameter
		filePrepender := func(s string) string { return "[Auto generated by spf13/cobra]: <>\n\n" }
		if err := cobradoc.GenMarkdownTreeCustom(cmd.Root(), "./docs", filePrepender, linkHandler); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(cmdGenerateDocs)
}
