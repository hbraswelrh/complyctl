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

	Config string
}

// BindFlags populate ComplyTime options from user-specified flags.
func (o *ComplyTime) BindFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.UserWorkspace, "workspace", "w", "./complytime", "workspace to use for artifact generation")
	fs.StringVarP(&o.Config, "config", "c", "./complytime/config.yml", "config for updating the assessment plan fields.")
}

// Added this on 04/07/2025 for Option 2 in editing configuration file
//type Configuration struct {
//	// Set config fields using viper and then update for the flags
//	Config string
//}

// Added this flag option 04/07/2025 to populate flags for the Config editing option
//func (o *ComplyTime) BindFlags(fs *pflag.FlagSet) {
//	fs.StringVarP(&o.Configuration, "config", "c", "config.yaml", "The config file to be leveraged for updating assessment plans")
//}

// ToPluginOptions returns global PluginOptions based on complytime Options.
func (o *ComplyTime) ToPluginOptions() complytime.PluginOptions {
	pluginOptions := complytime.NewPluginOptions()
	pluginOptions.Workspace = filepath.Clean(o.UserWorkspace)
	pluginOptions.Profile = o.FrameworkID
	return pluginOptions
}
