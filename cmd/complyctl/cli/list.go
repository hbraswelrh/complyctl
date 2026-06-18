// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/complytime/complyctl/internal/cache"
	"github.com/complytime/complyctl/internal/complytime"
	"github.com/complytime/complyctl/internal/policy"
	"github.com/complytime/complyctl/internal/terminal"
)

type listOptions struct {
	*Common
	policyID string
	cacheDir string
}

func listCmd(common *Common) *cobra.Command {
	o := &listOptions{
		Common: common,
	}
	cmd := &cobra.Command{
		Use:               "list [flags]",
		Short:             "List cached Gemara policies",
		SilenceUsage:      true,
		Example:           "complyctl list\n  complyctl list --policy-id nist-800-53-r5",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := o.validate(); err != nil {
				return err
			}
			if err := o.complete(); err != nil {
				return err
			}
			return o.run(cmd.Context())
		},
	}
	cmd.Flags().StringVar(&o.policyID, "policy-id", "", "Filter by policy ID")
	if err := cmd.RegisterFlagCompletionFunc("policy-id", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		logger.Error("Failed to register policy-id completion", "error", err)
	}
	return cmd
}

func (o *listOptions) validate() error {
	return nil
}

func (o *listOptions) complete() error {
	var err error
	o.cacheDir, err = complytime.ResolveCacheDir()
	if err != nil {
		return fmt.Errorf("failed to resolve cache directory: %w", err)
	}
	return nil
}

func (o *listOptions) run(_ context.Context) error {
	baseDir, err := o.ResolveWorkspace()
	if err != nil {
		return err
	}
	ws := complytime.NewWorkspace(baseDir)
	if err := ws.LoadAndValidate(); err != nil {
		return fmt.Errorf("failed to load workspace config: %w", err)
	}

	cfg := ws.Config()

	cacheMgr := cache.NewCache(o.cacheDir)
	loader := policy.NewLoader(cacheMgr)

	var rows [][]string
	for _, p := range cfg.Policies {
		eid := p.EffectiveID()
		if o.policyID != "" && eid != o.policyID {
			continue
		}

		ref, err := complytime.ParsePolicyRef(p.URL)
		if err != nil {
			return fmt.Errorf("invalid policy reference %q: %w", p.URL, err)
		}
		versions, _ := loader.GetCachedVersions(ref.Repository)

		var versionStr string
		if len(versions) > 0 {
			sort.Strings(versions)
			versionStr = strings.Join(versions, ", ")
		} else {
			versionStr = "-"
		}

		rows = append(rows, []string{eid, versionStr})
	}

	return printGemaraPolicyTable(o.Out, rows)
}

func printGemaraPolicyTable(w io.Writer, rows [][]string) error {
	sort.SliceStable(rows, func(i, j int) bool { return rows[i][0] < rows[j][0] })

	headers := []string{"POLICY ID", "VERSION"}
	terminal.ShowPlainTable(w, headers, rows)
	return nil
}
