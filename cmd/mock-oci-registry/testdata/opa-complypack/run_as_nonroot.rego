# SPDX-License-Identifier: Apache-2.0

package kubernetes.run_as_nonroot

import rego.v1

deny contains msg if {
	input.kind == "Deployment"
	containers := input.spec.template.spec.containers
	some container in containers
	not container.securityContext.runAsNonRoot
	msg := sprintf(
		"Container '%s' in Deployment '%s' must set securityContext.runAsNonRoot to true",
		[container.name, input.metadata.name],
	)
}
