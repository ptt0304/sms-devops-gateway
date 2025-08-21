package config

import (
	"encoding/json"
	"os"
)

type IgnoreConfig struct {
	Ignore []ClusterRule `json:"ignore"`
}

type ClusterRule struct {
	Cluster    string          `json:"cluster"`
	Namespaces []NamespaceRule `json:"namespaces"`
}

type NamespaceRule struct {
	Name string   `json:"name"`
	Pods []string `json:"pods"`
}

// LoadIgnoreConfig đọc file ignore-alert.json
func LoadIgnoreConfig(path string) (*IgnoreConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ignoreConfig IgnoreConfig
	err = json.Unmarshal(data, &ignoreConfig)
	if err != nil {
		return nil, err
	}

	return &ignoreConfig, nil
}
