package config

import (
	"encoding/json"
	"os"
	"time"
)

// TimeRange định nghĩa khoảng thời gian
type TimeRange struct {
	Start *time.Time `json:"start"`
	End   *time.Time `json:"end"`
}

// PodRule định nghĩa rule cho từng pod
type PodRule struct {
	Name string    `json:"name"`
	Time TimeRange `json:"time"`
}

// PodGroup định nghĩa nhóm pod trong namespace
type PodGroup struct {
	Name string    `json:"name"`
	Pods []string  `json:"pods"`
	Time TimeRange `json:"time"`
}

// NamespaceRule định nghĩa rule cho từng namespace
type NamespaceRule struct {
	Name      string     `json:"name"`
	Time      TimeRange  `json:"time"`
	PodGroups []PodGroup `json:"podGroups"`
	Pods      []PodRule  `json:"pods"`
}

// NamespaceGroup định nghĩa nhóm namespace trong cluster
type NamespaceGroup struct {
	Name       string    `json:"name"`
	Namespaces []string  `json:"namespaces"`
	Time       TimeRange `json:"time"`
}

// ClusterRule định nghĩa rule cho từng cluster
type ClusterRule struct {
	Cluster         string           `json:"cluster"`
	Time            TimeRange        `json:"time"`
	NamespaceGroups []NamespaceGroup `json:"namespaceGroups"`
	Namespaces      []NamespaceRule  `json:"namespaces"`
}

// ClusterGroup định nghĩa nhóm cluster
type ClusterGroup struct {
	Name     string    `json:"name"`
	Clusters []string  `json:"clusters"`
	Time     TimeRange `json:"time"`
}

// IgnoreConfig định nghĩa toàn bộ cấu hình ignore
type IgnoreConfig struct {
	ClusterGroups []ClusterGroup `json:"clusterGroups"`
	Ignore        []ClusterRule  `json:"ignore"`
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
