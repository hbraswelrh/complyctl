// SPDX-License-Identifier: Apache-2.0

package complypack

const (
	// MediaTypeArtifact is the OCI artifactType for ComplyPack artifacts.
	// Used as the second parameter to oras.PackManifest().
	MediaTypeArtifact = "application/vnd.complypack.artifact.v1"

	// MediaTypeConfig is the OCI config layer media type.
	// Contains evaluator-id, version, and optional provenance.
	MediaTypeConfig = "application/vnd.complypack.config.v1+json"

	// MediaTypeContent is the OCI content layer media type.
	// Contains opaque policy content (e.g., OPA bundle tarball).
	MediaTypeContent = "application/vnd.complypack.content.v1.tar+gzip"
)
