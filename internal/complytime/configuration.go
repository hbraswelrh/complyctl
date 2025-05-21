// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/oscal-compass/compliance-to-policy-go/v2/framework"
	"github.com/oscal-compass/compliance-to-policy-go/v2/framework/actions"
	"github.com/oscal-compass/oscal-sdk-go/models"
	"github.com/oscal-compass/oscal-sdk-go/models/components"
	"github.com/oscal-compass/oscal-sdk-go/validation"

	"github.com/complytime/complytime/internal/complytime/plan"
)

const (
	compDefSuffix       = "component-definition.json"
	ApplicationDir      = "complytime"
	PluginDir           = "plugins"
	BundlesDir          = "bundles"
	ControlsDir         = "controls"
	DataRootDir         = "/usr/share"
	PluginBinaryRootDir = "/usr/libexec/"
)

// ErrNoComponentDefinitionsFound returns an error indicated the supplied directory
// does not contain component definitions that are detectable by complytime.
var ErrNoComponentDefinitionsFound = errors.New("no component definitions found")

// ApplicationDirectory represents the directories that make up
// the complytime application directory.
type ApplicationDirectory struct {
	// appDir is the top-level directory
	appDir string
	// pluginDir contains all complytime binary plugins.
	pluginDir string
	// pluginManifestDir contains all complytime plugin manifests.
	pluginManifestDir string
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
	// When running local built complytime for development
	if os.Getenv("COMPLYTIME_DEV_MODE") == "1" {
		return newApplicationDirectory(xdg.DataHome, create)
	} else {
		return newApplicationDirectory(DataRootDir, false)
	}
}

// newApplicationDirectory returns a new ApplicationDirectory with the
// given root directory. Creation of the directories is optional using the
// `create` input. If the application directories exist, this will not overwrite what is
// existing.
func newApplicationDirectory(rootDir string, create bool) (ApplicationDirectory, error) {
	applicationDir := ApplicationDirectory{
		appDir: filepath.Join(rootDir, ApplicationDir),
	}
	// Drop-in configuration to be supported in CPLYTM-716
	applicationDir.pluginManifestDir = filepath.Join(applicationDir.appDir, PluginDir)
	if rootDir == DataRootDir {
		applicationDir.pluginDir = filepath.Join(PluginBinaryRootDir, ApplicationDir, PluginDir)
	} else {
		applicationDir.pluginDir = applicationDir.pluginManifestDir
	}
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

// PluginManifestDir returns the directory containing plugin manifests.
// definition.
func (a ApplicationDirectory) PluginManifestDir() string {
	return a.pluginManifestDir
}

// Dirs returns all directories in the ApplicationDirectory.
func (a ApplicationDirectory) Dirs() []string {
	return []string{
		a.appDir,
		a.pluginDir,
		a.pluginManifestDir,
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
func Config(a ApplicationDirectory) (*framework.C2PConfig, error) {
	cfg := framework.DefaultConfig()
	cfg.PluginDir = a.PluginDir()
	cfg.PluginManifestDir = a.PluginManifestDir()
	return cfg, nil
}

// ActionsContextFromPlan returns a new actions.InputContext from a given OSCAL AssessmentPlan.
func ActionsContextFromPlan(assessmentPlan *oscalTypes.AssessmentPlan) (*actions.InputContext, error) {
	if assessmentPlan.AssessmentAssets.Components == nil {
		return nil, errors.New("assessment plan has no assessment components")
	}
	var allComponents []components.Component
	for _, component := range *assessmentPlan.AssessmentAssets.Components {
		compAdapter := components.NewSystemComponentAdapter(component)
		allComponents = append(allComponents, compAdapter)
	}
	inputContext, err := actions.NewContext(allComponents)
	if err != nil {
		return nil, fmt.Errorf("error generating context from plan %s: %w", assessmentPlan.Metadata.Title, err)
	}
	apSettings, err := plan.Settings(assessmentPlan)
	if err != nil {
		return nil, fmt.Errorf("cannot extract settings from plan %s: %w", assessmentPlan.Metadata.Title, err)
	}
	inputContext.Settings = apSettings
	return inputContext, nil
}
