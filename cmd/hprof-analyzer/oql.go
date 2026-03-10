package main

import (
	"fmt"
	"os"

	"github.com/modbender/hprof-analyzer/internal/index"
	"github.com/modbender/hprof-analyzer/internal/oql"
	"github.com/modbender/hprof-analyzer/internal/output"

	"github.com/spf13/cobra"
)

func newOQLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   `oql <file.hprof> "<query>"`,
		Short: "Execute an OQL query against the heap dump",
		Long: `Execute a SQL-like OQL (Object Query Language) query against the heap dump.

Examples:
  hprof-analyzer oql dump.hprof "SELECT @class, @shallowSize FROM java.util.HashMap"
  hprof-analyzer oql dump.hprof "SELECT @class, count(*) FROM instanceof java.util.AbstractMap GROUP BY @class ORDER BY count DESC LIMIT 10"
  hprof-analyzer oql dump.hprof "SELECT * FROM java.lang.String WHERE @shallowSize > 1024"`,
		Args: cobra.ExactArgs(2),
		RunE: runOQL,
	}
}

func runOQL(cmd *cobra.Command, args []string) error {
	stmt, err := oql.Parse(args[1])
	if err != nil {
		return fmt.Errorf("parsing OQL: %w", err)
	}

	idx, err := index.EnsureIndexed(args[0])
	if err != nil {
		return err
	}

	headers, results, err := oql.Eval(stmt, idx)
	if err != nil {
		return fmt.Errorf("evaluating OQL: %w", err)
	}

	w, err := getOutput()
	if err != nil {
		return err
	}
	if w != os.Stdout {
		defer w.Close()
	}

	fmtr, err := output.NewFormatter(w, outputFmt)
	if err != nil {
		return err
	}

	fmtr.WriteHeader(headers)
	for _, r := range results {
		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = r.Values[h]
		}
		fmtr.WriteRow(row)
	}
	return fmtr.Flush()
}
