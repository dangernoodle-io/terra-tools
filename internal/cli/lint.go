package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"dangernoodle.io/terra-tools/internal/lint/report"
	"dangernoodle.io/terra-tools/internal/lint/validate"
	"dangernoodle.io/terra-tools/internal/output"
)

var (
	lintDirFlag       string
	lintRecursiveFlag bool
)

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint terragrunt stack configs",
	RunE:  runLint,
}

func init() {
	lintCmd.Flags().StringVarP(&lintDirFlag, "dir", "d", "", "Directory to lint (default: current directory)")
	lintCmd.Flags().BoolVarP(&lintRecursiveFlag, "recursive", "r", false, "Recursively lint all subdirectories")
}

func runLint(cmd *cobra.Command, args []string) error {
	dir := lintDirFlag
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("lint: get working directory: %w", err)
		}
		dir = cwd
	}

	var errs []validate.Error
	var err error

	if lintRecursiveFlag {
		errs, err = validate.WalkDir(dir)
	} else {
		errs, err = validate.Dir(dir)
	}
	if err != nil {
		return err
	}

	if len(errs) > 0 {
		report.Print(os.Stdout, errs)
		return fmt.Errorf("lint: found %d issue(s)", len(errs))
	}

	output.Success("No issues found")
	return nil
}
