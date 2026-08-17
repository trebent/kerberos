package cmd

import (
	"encoding/json"
	"fmt"
)

// buildConnectorJSON generates a minimal connector.json.
func buildConnectorJSON(opts *connectorOptions) ([]byte, error) {
	origin := opts.corsOrigin
	if origin == "" {
		origin = defaultConnectorTarget
	}

	root := map[string]any{
		"origins": map[string]any{
			"allowedOrigins": []string{origin},
		},
		"persistence": buildPersistenceSection(opts.persistenceMode),
	}

	content, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal connector config: %w", err)
	}

	return content, nil
}
