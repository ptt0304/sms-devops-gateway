package config

import (
	"encoding/json"
	"os"
	"time"
)

type TimeRange struct {
	Start *time.Time `json:"start"`
	End   *time.Time `json:"end"`
}

type PodRule struct {
	Name string    `json:"name"`
	Time TimeRange `json:"time"`
}

type NamespaceRule struct {
	Name string    `json:"name"`
	Time TimeRange `json:"time"`
	Pods []PodRule `json:"pods"`
}

type ClusterRule struct {
	Cluster    string          `json:"cluster"`
	Time       TimeRange       `json:"time"`
	Namespaces []NamespaceRule `json:"namespaces"`
}

type IgnoreConfig struct {
	Ignore []ClusterRule `json:"ignore"`
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