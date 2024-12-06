package util

import (
	"go.temporal.io/sdk/workflow"
)

// GetVersion retrieves the version for a workflow change, allowing for an optional override.
// - versionOverride: Map containing override versions for specific change IDs.
// - changeID: A unique identifier for a workflow change.
// - minSupported: The minimum supported version.
// - maxSupported: The maximum supported version.
func GetVersion(ctx workflow.Context, versionOverride map[string]int, changeID string, minSupported, maxSupported workflow.Version) workflow.Version {
	if overriddenVersion, ok := versionOverride[changeID]; ok {
		// Register the marker with the overridden version
		workflow.GetVersion(ctx, changeID, workflow.Version(overriddenVersion), workflow.Version(maxSupported))
		return workflow.Version(overriddenVersion)
	}

	// Fallback to Temporal's default GetVersion logic
	return workflow.GetVersion(ctx, changeID, minSupported, maxSupported)
}
