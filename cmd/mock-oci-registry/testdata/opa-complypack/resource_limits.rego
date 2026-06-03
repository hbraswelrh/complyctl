# SPDX-License-Identifier: Apache-2.0

package kubernetes.resource_limits

import rego.v1

deny contains msg if {
	input.kind == "Deployment"
	containers := input.spec.template.spec.containers
	some container in containers
	not container.resources.limits
	msg := sprintf(
		"Container '%s' in Deployment '%s' must define resource limits",
		[container.name, input.metadata.name],
	)
}
