// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-2"
	"github.com/oscal-compass/compliance-to-policy-go/v2/framework/config"
	"github.com/oscal-compass/oscal-sdk-go/models"
	"github.com/oscal-compass/oscal-sdk-go/validation"
)

const (
	compDefSuffix  = "component-definition.json"
	ApplicationDir = "complytime"
	PluginDir      = "plugins"
	BundlesDir     = "bundles"
	ControlsDir    = "controls"
)

// ErrNoComponentDefinitionsFound returns an error indicated the supplied directory
// does not contain component definitions that are detectable by complytime.
var ErrNoComponentDefinitionsFound = errors.New("no component definitions found")

// ApplicationDirectory represents the directories that make up
// the complytime application directory.
type ApplicationDirectory struct {
	// appDir is the top-level directory
	appDir string
	// pluginDir is below the appDir and contains
	// all complytime plugins.
	pluginDir string
	// bundleDir contains all the detectable component definitions
	bundleDir string
	// controlDir contains all OSCAL control layer models.
	controlDir string
}

// NewApplicationDirectory returns a new ApplicationDirectory.
//
// Creation of the directories is optional using the `create` input.
// If the application directories exist, this will not overwrite what is
// existing.
func NewApplicationDirectory(create bool) (ApplicationDirectory, error) {
	return newApplicationDirectory(xdg.ConfigHome, create)
}

// newApplicationDirectory returns a new ApplicationDirectory with the
// given root directory. Creation of the directories is optional using the
// `create` input. If the application directories exist, this will not overwrite what is
// existing.
func newApplicationDirectory(rootDir string, create bool) (ApplicationDirectory, error) {
	applicationDir := ApplicationDirectory{
		appDir: filepath.Join(rootDir, ApplicationDir),
	}
	applicationDir.pluginDir = filepath.Join(applicationDir.appDir, PluginDir)
	applicationDir.bundleDir = filepath.Join(applicationDir.appDir, BundlesDir)
	applicationDir.controlDir = filepath.Join(applicationDir.appDir, ControlsDir)
	if create {
		return applicationDir, applicationDir.create()
	}
	return applicationDir, nil
}

// create creates the application directories if they do not exist.
func (a ApplicationDirectory) create() error {
	for _, dir := range a.Dirs() {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("unable to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// AppDir returns the top-level application directory.
func (a ApplicationDirectory) AppDir() string {
	return a.appDir
}

// PluginDir returns the plugin directory below the AppDir.
func (a ApplicationDirectory) PluginDir() string {
	return a.pluginDir
}

// BundleDir returns the bundle directory containing the component
// definition.
func (a ApplicationDirectory) BundleDir() string {
	return a.bundleDir
}

// ControlDir returns the directory containing control layer OSCAL artifacts.
func (a ApplicationDirectory) ControlDir() string { return a.controlDir }

// Dirs returns all directories in the ApplicationDirectory.
func (a ApplicationDirectory) Dirs() []string {
	return []string{
		a.appDir,
		a.pluginDir,
		a.bundleDir,
		a.controlDir,
	}
}

// FindComponentDefinitions locates all the OSCAL Component Definitions in the
// given `bundles` directory that meet the defined naming scheme.
//
// The defined scheme is $COMPONENT-NAME-component-definition.json.
func FindComponentDefinitions(bundleDir string, validator validation.Validator) ([]oscalTypes.ComponentDefinition, error) {
	items, err := os.ReadDir(bundleDir)
	if err != nil {
		return nil, fmt.Errorf("unable to read bundle directory %s: %w", bundleDir, err)
	}

	var compDefBundles []oscalTypes.ComponentDefinition
	for _, item := range items {
		if !strings.HasSuffix(item.Name(), compDefSuffix) {
			continue
		}
		compDefPath := filepath.Join(bundleDir, item.Name())
		compDefPath = filepath.Clean(compDefPath)
		file, err := os.Open(compDefPath)
		if err != nil {
			return nil, err
		}
		definition, err := models.NewComponentDefinition(file, validator)
		if err != nil {
			return nil, err
		}
		if definition == nil {
			return nil, fmt.Errorf("could not load component definition from %s", compDefPath)
		}
		compDefBundles = append(compDefBundles, *definition)
	}
	if len(compDefBundles) == 0 {
		return nil, fmt.Errorf("directory %s: %w", bundleDir, ErrNoComponentDefinitionsFound)
	}
	return compDefBundles, nil
}

// Config creates a new C2P config for the ComplyTime CLI to use to configure
// the plugin manager.
func Config(a ApplicationDirectory, validator validation.Validator) (*config.C2PConfig, error) {
	cfg := config.DefaultConfig()
	cfg.PluginDir = a.PluginDir()

	compDefBundles, err := FindComponentDefinitions(a.BundleDir(), validator)
	if err != nil {
		return cfg, fmt.Errorf("unable to create configuration: %w", err)
	}
	cfg.ComponentDefinitions = compDefBundles
	return cfg, nil
}
