// SPDX-License-Identifier: Apache-2.0

package config

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/hashicorp/go-hclog"
)

const (
	PluginDir      string = "openscap"
	DatastreamsDir string = "/usr/share/xml/scap/ssg/content"
	SystemInfoFile string = "/etc/os-release"
)

type Config struct {
	Files struct {
		Workspace  string `config:"workspace"`
		Datastream string `config:"datastream"`
		Results    string `config:"results"`
		ARF        string `config:"arf"`
		Policy     string `config:"policy"`
	}
	Parameters struct {
		Profile string `config:"profile"`
	}
}

// NewConfig creates a new, empty Config.
func NewConfig() *Config {
	return &Config{}
}

// LoadSettings sets the values in the Config from a given config map and
// performs validation.
func (c *Config) LoadSettings(config map[string]string) error {
	filesVal := reflect.ValueOf(&c.Files).Elem()
	if err := setConfigStruct(filesVal, config); err != nil {
		return err
	}
	paramVal := reflect.ValueOf(&c.Parameters).Elem()
	if err := setConfigStruct(paramVal, config); err != nil {
		return err
	}
	return c.validate()
}

func (c *Config) validate() error {
	// String values to sanitize
	inputValues := []*string{
		&c.Files.Policy,
		&c.Files.Results,
		&c.Files.ARF,
		&c.Parameters.Profile,
	}

	for _, inputValue := range inputValues {
		sanitized, err := SanitizeInput(*inputValue)
		if err != nil {
			return err
		}
		*inputValue = sanitized
	}

	cleanDsPath, err := SanitizePath(c.Files.Datastream)
	if err != nil {
		return err
	}

	// if a Datastream path is not defined in plugin manifest, it will be set
	// to the current directory after SanitizePath.
	if cleanDsPath == "." {
		matchingDsFile, err := findMatchingDatastream()
		if err != nil {
			return err
		}
		c.Files.Datastream = matchingDsFile
	}

	_, err = validatePath(c.Files.Datastream, false)
	if err != nil {
		return fmt.Errorf("invalid datastream path: %s: %w", c.Files.Datastream, err)
	}

	isXML, err := IsXMLFile(c.Files.Datastream)
	if err != nil || !isXML {
		return fmt.Errorf("invalid datastream file: %s: %w", c.Files.Datastream, err)
	}

	if err := defineFilesPaths(c); err != nil {
		return err
	}
	return nil
}

func SanitizeInput(input string) (string, error) {
	safePattern := regexp.MustCompile(`^[a-zA-Z0-9-_.]+$`)
	if !safePattern.MatchString(input) {
		return "", fmt.Errorf("input contains unexpected characters: %s", input)
	}
	return input, nil
}

func expandPath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed to identify current user: %w", err)
		}
		homeDir := usr.HomeDir
		// Replace "~" with the home directory
		return filepath.Join(homeDir, path[1:]), nil
	}
	return path, nil
}

func validatePath(path string, shouldBeDir bool) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to confirm path existence: %w", err)
	}

	if shouldBeDir && !stat.IsDir() {
		return "", fmt.Errorf("expected a directory, but found a file at path: %s", path)
	}
	if !shouldBeDir && stat.IsDir() {
		return "", fmt.Errorf("expected a file, but found a directory at path: %s", path)
	}

	return path, nil
}

func SanitizePath(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	expandedPath, err := expandPath(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to expand path: %w", err)
	}
	return expandedPath, nil
}

func IsXMLFile(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return true, nil
			}
			return false, fmt.Errorf("invalid XML file %s: %w", filePath, err)
		}
	}
}

func ensureDirectory(path string) error {
	_, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		err := os.MkdirAll(path, 0750)
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		hclog.Default().Info("Directory created", "path", path)
	} else if err != nil {
		return fmt.Errorf("error checking directory: %w", err)
	}

	return nil
}

func ensureWorkspace(cfg *Config) (map[string]string, error) {
	workspacePath, err := SanitizePath(cfg.Files.Workspace)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize workspace path %s: %w", cfg.Files.Workspace, err)
	}

	workspace, err := validatePath(workspacePath, true)
	if err != nil {
		hclog.Default().Info("Informed workspace was not found. It will be created.")
		workspace = workspacePath
	}

	directories := map[string]string{
		"workspace":  workspace,
		"pluginDir":  filepath.Join(workspace, PluginDir),
		"policyDir":  filepath.Join(workspace, PluginDir, "policy"),
		"resultsDir": filepath.Join(workspace, PluginDir, "results"),
	}

	for key, dir := range directories {
		if err := ensureDirectory(dir); err != nil {
			return nil, fmt.Errorf("failed to ensure directory %s (%s): %w", dir, key, err)
		}
	}

	return directories, nil
}

func defineFilesPaths(cfg *Config) error {
	directories, err := ensureWorkspace(cfg)
	if err != nil {
		return err
	}

	cfg.Files.Policy = filepath.Join(directories["policyDir"], cfg.Files.Policy)
	cfg.Files.Results = filepath.Join(directories["resultsDir"], cfg.Files.Results)
	cfg.Files.ARF = filepath.Join(directories["resultsDir"], cfg.Files.ARF)

	return nil
}

// setConfigStruct populates struct fields with matching tags to values
// in a given config map.
func setConfigStruct(val reflect.Value, config map[string]string) error {
	t := val.Type()
	for i := 0; i < val.NumField(); i++ {
		fieldType := t.Field(i)
		key := fieldType.Tag.Get("config")
		value, ok := config[key]
		// if datastream is not set in manifest file, plugin will try to determine
		// and validate the datastream path later based on system information.
		if !ok && key != "datastream" {
			return fmt.Errorf("missing configuration value for option %q (field: %s)", key, fieldType.Name)
		}

		fieldVal := val.Field(i)
		fieldVal.SetString(value)
	}
	return nil
}

// GetDistroIdsAndVersions returns a slice of allowable distribution IDs and allowable versions of the system
// based on information from SystemInfoFile.
// Example return values: ["centos", "rhel", "fedora"], ["9", "95"]
func getDistroIdsAndVersions() ([]string, []string, error) {
	file, err := os.Open(SystemInfoFile)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	// Like ["rhel", "fedora", "centos"]
	var ids []string
	// Like "9.5"
	var versionID string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id := strings.Trim(strings.Split(line, "=")[1], `"`)
			ids = append(ids, id)
		} else if strings.HasPrefix(line, "VERSION_ID=") {
			versionID = strings.Trim(strings.Split(line, "=")[1], `"`)
		} else if strings.HasPrefix(line, "ID_LIKE=") {
			altIdString := strings.Trim(strings.Split(line, "=")[1], `"`)
			altIds := strings.Split(altIdString, " ")
			ids = append(ids, altIds...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	if ids != nil && versionID != "" {
		// Extract major version (e.g., 9 from 9.5)
		majorVersion := strings.Split(versionID, ".")[0]
		// Also keep the full version without dots (e.g., "95" from "9.5")
		fullVersion := strings.ReplaceAll(versionID, ".", "")

		return ids, []string{majorVersion, fullVersion}, nil
	}

	return nil, nil, fmt.Errorf("could not determine distribution and version based on %s", SystemInfoFile)
}

func findMatchingDatastream() (string, error) {
	distroIds, distroVersions, err := getDistroIdsAndVersions()
	if err != nil {
		return "", err
	}

	// The scap-security-guide package uses datastream filenames like "ssg-rhel9-ds.xml"
	// where rhel is ID and 9 is the VERSION_ID

	var foundFile string

	err = filepath.Walk(DatastreamsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			for id := range distroIds {
				for version := range distroVersions {
					pattern := fmt.Sprintf("ssg-%s%s-ds.xml", distroIds[id], distroVersions[version])
					if info.Name() == pattern {
						foundFile = path
						return filepath.SkipDir
					}
				}
				if distroVersions == nil {
					// In case of non-versioned or rolling release
					pattern := fmt.Sprintf("ssg-%s-ds.xml", distroIds[id])
					if info.Name() == pattern {
						foundFile = path
						return filepath.SkipDir
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		return "", err
	}
	if foundFile != "" {
		return foundFile, nil
	}

	return "", fmt.Errorf("could not determine a datastream file for a system with ids: %v and versions: %v", distroIds, distroVersions)
}
