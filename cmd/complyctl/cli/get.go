// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/registry/remote/auth"

	"github.com/complytime/complyctl/internal/cache"
	"github.com/complytime/complyctl/internal/complytime"
	"github.com/complytime/complyctl/internal/registry"
)

type getOptions struct {
	*Common
	timeout  time.Duration
	cacheDir string
}

func getCmd(common *Common) *cobra.Command {
	o := &getOptions{
		Common: common,
	}
	cmd := &cobra.Command{
		Use:               "get [flags]",
		Short:             "Fetch new/modified policies from OCI registry and update cache",
		SilenceUsage:      true,
		Example:           "complyctl get",
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
	cmd.Flags().DurationVarP(&o.timeout, "timeout", "t", complytime.DefaultCommandTimeout, "Maximum time for the get operation (e.g. 5m, 10m, 1h)")
	return cmd
}

func (o *getOptions) validate() error {
	return nil
}

func (o *getOptions) complete() error {
	var err error
	o.cacheDir, err = complytime.ResolveCacheDir()
	if err != nil {
		return fmt.Errorf("failed to resolve cache directory: %w", err)
	}
	return nil
}

func (o *getOptions) run(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	cfg, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	return o.syncPolicies(ctx, cfg)
}

func (o *getOptions) syncPolicies(ctx context.Context, cfg *complytime.WorkspaceConfig) error {
	cacheMgr := cache.NewCache(o.cacheDir)

	state, err := cache.LoadState(o.cacheDir)
	if err != nil {
		logger.Error("Cache state load failed", "cache_dir", o.cacheDir, "error", err)
		return fmt.Errorf("failed to load cache state: %w", err)
	}

	credFunc, err := registry.NewCredentialFunc()
	if err != nil {
		logger.Error("Credential resolution failed", "error", err)
		return fmt.Errorf("authentication setup failed: %w", err)
	}

	return syncAllPolicies(ctx, cacheMgr, state, credFunc, cfg.Policies, o.cacheDir)
}

func syncAllPolicies(ctx context.Context, cacheMgr *cache.Cache, state *cache.State, credFunc auth.CredentialFunc, policies []complytime.PolicyEntry, cacheDir string) error {
	logger.Info("Starting policy synchronization", "policy_count", len(policies))

	total := len(policies)
	for i, entry := range policies {
		if err := syncSinglePolicy(ctx, cacheMgr, state, credFunc, entry, i+1, total, cacheDir); err != nil {
			return err
		}
	}

	logger.Info("Synchronization completed", "synced", total, "total", total)
	fmt.Fprintln(os.Stderr, "Synchronization completed.")
	return nil
}

func syncSinglePolicy(ctx context.Context, cacheMgr *cache.Cache, state *cache.State, credFunc auth.CredentialFunc, entry complytime.PolicyEntry, index, total int, cacheDir string) error {
	ref := complytime.ParsePolicyRef(entry.URL)
	version := ref.Version

	// Dispatch based on scheme: local OCI Layout vs. remote registry.
	// For local sources, the cache key is the effective ID (not the path).
	// For remote sources, the cache key is the repository path.
	var source cache.PolicySource
	var syncPolicyID string
	if ref.Registry == complytime.OCILayoutScheme {
		source = cache.NewLocalSource(ref.Repository)
		syncPolicyID = entry.EffectiveID()
	} else {
		client := registry.NewClient(ref.Registry, credFunc)
		source = cache.NewRegistrySource(client)
		syncPolicyID = ref.Repository

		if version == "" {
			version = resolveLatestVersion(ctx, client, ref.Repository, entry.EffectiveID())
		}
	}

	sync := cache.NewSync(cacheMgr, state, source)

	fmt.Fprintf(os.Stderr, "Syncing policy %d/%d: %s... ", index, total, entry.EffectiveID())
	logger.Info("Syncing policy", "policy", syncPolicyID, "version", version)
	if err := sync.SyncPolicy(ctx, syncPolicyID, version); err != nil {
		fmt.Fprintln(os.Stderr, "failed")
		suggestMsg := suggestCachedPolicyIDs(cacheDir, syncPolicyID)
		logger.Error("Policy sync failed", "policy", syncPolicyID, "error", err)
		return fmt.Errorf("failed to sync policy %s: %w%s", syncPolicyID, err, suggestMsg)
	}
	fmt.Fprintln(os.Stderr, "done")
	logger.Info("Policy synced", "policy", entry.EffectiveID())
	return nil
}

func resolveLatestVersion(ctx context.Context, client *registry.Client, repository, policyID string) string {
	logger.Info("Resolving latest version", "policy", policyID)
	_, resolvedVersion, resolveErr := client.DefinitionVersion(ctx, repository)
	if resolveErr != nil {
		logger.Warn("Version resolution failed, falling back to 'latest'",
			"policy", policyID, "error", resolveErr)
		return "latest"
	}
	logger.Info("Resolved version", "policy", policyID, "version", resolvedVersion)
	return resolvedVersion
}

func suggestCachedPolicyIDs(cacheDir, failedPolicyID string) string {
	state, err := cache.LoadState(cacheDir)
	if err != nil || len(state.Policies) == 0 {
		return ""
	}
	cached := make([]string, 0, len(state.Policies))
	for id := range state.Policies {
		if id != failedPolicyID {
			cached = append(cached, id)
		}
	}
	if len(cached) == 0 {
		return ""
	}
	return fmt.Sprintf(" (cached policies: %v)", cached)
}
