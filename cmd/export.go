package cmd

import (
	"fmt"
	"log"

	"github.com/metal-automata/agent/internal/firmware/outofband"
	"github.com/metal-automata/agent/internal/firmware/runner"
	"github.com/spf13/cobra"

	"github.com/emicklei/dot"
)

var cmdExportFlowDiagram = &cobra.Command{
	Use:   "export-diagram",
	Short: "Export mermaidjs flowchart for firmware task transitions",
	Run: func(cmd *cobra.Command, _ []string) {
		g := runner.Graph()
		if err := outofband.GraphSteps(cmd.Context(), g); err != nil {
			log.Fatal(err)
		}

		fmt.Println(dot.MermaidGraph(g, dot.MermaidTopDown))
	},
}

func init() {
	rootCmd.AddCommand(cmdExportFlowDiagram)
}
