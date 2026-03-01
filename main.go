// Copyright (c) arkade author(s) 2022. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"os"

	"github.com/alexellis/gha-bump/pkg/ghabump"
	"github.com/spf13/cobra"
)

func main() {
	var target string
	var verbose bool
	var write bool

	var rootCmd = &cobra.Command{
		Use:   "gha-bump",
		Short: "Upgrade GitHub Actions workflow files to latest major versions",
		Long: `Upgrade actions in GitHub Actions workflow files to the latest major version.

Processes all workflow YAML files in .github/workflows/ or a single file.
Only bumps major versions (e.g. actions/checkout@v3 to actions/checkout@v4).
`,
		Example: `  # Upgrade all workflows in the current directory
  gha-bump

  # Upgrade a single workflow file
  gha-bump -f .github/workflows/build.yaml

  # Dry-run mode, don't write changes
  gha-bump --write=false`,
	}

	rootCmd.Flags().StringP("file", "f", ".", "Path to workflow file or directory")
	rootCmd.Flags().BoolP("verbose", "v", true, "Verbose output")
	rootCmd.Flags().BoolP("write", "w", true, "Write the updated values back to the file")

	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		target, _ = cmd.Flags().GetString("file")
		verbose, _ = cmd.Flags().GetBool("verbose")
		write, _ = cmd.Flags().GetBool("write")

		return ghabump.Run(ghabump.RunOptions{
			Target:  target,
			Verbose: verbose,
			Write:   write,
		})
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
