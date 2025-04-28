// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	oscalTypes "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-2"
	"github.com/oscal-compass/oscal-sdk-go/models"
	"github.com/oscal-compass/oscal-sdk-go/validation"
)

// The assessment results md file needs the catalog title information
// LoadCatalogSource returns an OSCAL catalogs from a given application directory and a found catalog source.
func LoadCatalogSource(appDir ApplicationDirectory, catalogSource string, validator validation.Validator) (*oscalTypes.Catalog, error) {
	sourceFile, err := findControlSource(appDir, catalogSource)
	if err != nil {
		return nil, err
	}
	defer sourceFile.Close()
	return models.NewCatalog(sourceFile, validator)
}
