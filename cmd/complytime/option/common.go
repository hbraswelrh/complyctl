// SPDX-License-Identifier: Apache-2.0

package option

import (
	"io"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/complytime/complytime/internal/complytime"
)

// Common options for the ComplyTime CLI.
type Common struct {
	Debug bool
	Output
}

// Output options for
type Output struct {
	// Out think, os.Stdout
	Out io.Writer
	// ErrOut think, os.Stderr
	ErrOut io.Writer
}

// BindFlags populate Common options from user-specified flags.
func (o *Common) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&o.Debug, "debug", "d", false, "output debug logs")
}

// ComplyTime options are configurations needed for the ComplyTime CLI to run.
// They are less generic the Common options and would only be used in a subset of
// commands.
type ComplyTime struct {
	// UserWorkspace is the location where all output artifacts should be written. This is set
	// by flags.
	UserWorkspace string
	// FrameworkID representing the compliance framework identifier associated with the artifacts in the workspace.
	// It is set by workspace state or command positional arguments.
	FrameworkID string
}

// BindFlags populate ComplyTime options from user-specified flags.
func (o *ComplyTime) BindFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.UserWorkspace, "workspace", "w", "./complytime", "workspace to use for artifact generation")
}

// ToPluginOptions returns global PluginOptions based on complytime Options.
func (o *ComplyTime) ToPluginOptions() complytime.PluginOptions {
	pluginOptions := complytime.NewPluginOptions()
	pluginOptions.Workspace = filepath.Clean(o.UserWorkspace)
	pluginOptions.Profile = o.FrameworkID
	return pluginOptions
}
