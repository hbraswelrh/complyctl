// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/oscal-compass/compliance-to-policy-go/v2/framework"
	"github.com/oscal-compass/compliance-to-policy-go/v2/policy"
)

// PluginOptions defines global options all complytime plugins should
// support.
type PluginOptions struct {
	// Workspace is the location where all
	// plugin outputs should be written.
	Workspace string `config:"workspace"`
	// Profile is the compliance profile that the plugin should use for
	// pre-defined policy groups.
	Profile string `config:"profile"`
}

// NewPluginOptions created a new PluginOptions struct.
func NewPluginOptions() PluginOptions {
	return PluginOptions{}
}

// Validate ensure the required plugin options are set.
func (p PluginOptions) Validate() error {
	// TODO[jpower432]: If these options grow, using third party
	// validation through struct tags could be simpler if the validation
	// logic gets more complex.
	if p.Workspace == "" {
		return errors.New("workspace must be set")
	}
	if p.Profile == "" {
		return errors.New("profile must be set")
	}
	return nil
}

// ToMap transforms the PluginOption struct into a map that can be consumed
// by the C2P Plugin Manager.
func (p PluginOptions) ToMap() map[string]string {
	selections := make(map[string]string)
	val := reflect.ValueOf(p)
	t := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := t.Field(i)
		key := fieldType.Tag.Get("config")
		selections[key] = field.String()
	}
	return selections
}

// Plugins launches and configures plugins with the given complytime global options. This function returns the plugin map with the
// launched plugins, a plugin cleanup function, and an error. The cleanup function should be used if it is not nil.
func Plugins(manager *framework.PluginManager, selections PluginOptions) (map[string]policy.Provider, func(), error) {
	manifests, err := manager.FindRequestedPlugins()
	if err != nil {
		return nil, nil, err
	}

	getSelections := func(_ string) map[string]string {
		return selections.ToMap()
	}

	if err := selections.Validate(); err != nil {
		return nil, nil, fmt.Errorf("failed plugin config validation: %w", err)
	}
	plugins, err := manager.LaunchPolicyPlugins(manifests, getSelections)
	// Plugin subprocess has now been launched; cleanup always required below
	if err != nil {
		return nil, manager.Clean, err
	}
	return plugins, manager.Clean, nil
}
