package cmd

import (
	"encoding/json"
	"fmt"
)

// buildConnectorJSON generates a minimal connector.json.
func buildConnectorJSON(opts *connectorOptions) ([]byte, error) {
	origins := map[string]any{}
	if opts.allowAllOrigins {
		origins["allowAll"] = true
	} else {
		origins["denyAll"] = true
	}

	root := map[string]any{
		"origins":     origins,
		"persistence": buildPersistenceSection(opts.persistenceMode),
	}

	content, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connector config: %w", err)
	}

	return content, nil
}
