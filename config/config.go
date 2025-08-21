package config

import (
	"os"
	"strings"
	"encoding/json"
)

type Receiver struct {
	Name   string            `json:"name"`
	Mobiles []string          `json:"-"`     // danh sách số điện thoại đã tách
	Mobile  string            `json:"mobile"` // chuỗi raw từ json
	Match  map[string]string `json:"match"`
}

type DefaultReceiver struct {
	Mobiles []string `json:"-"`
	Mobile  string   `json:"mobile"`
}

type Config struct {
	Receivers       []Receiver     `json:"receiver"`
	DefaultReceiver DefaultReceiver `json:"default_receiver"`
}

// LoadConfig loads the json config file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.Normalize()
	return &cfg, nil
}

// Normalize parses raw mobile strings into string slices
func (c *Config) Normalize() {
	for i := range c.Receivers {
		c.Receivers[i].Mobiles = parseMobiles(c.Receivers[i].Mobile)
	}
	c.DefaultReceiver.Mobiles = parseMobiles(c.DefaultReceiver.Mobile)
}

// parseMobiles splits and trims a comma-separated mobile string
func parseMobiles(mobileString string) []string {
	parts := strings.Split(mobileString, ",")
	var mobiles []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			mobiles = append(mobiles, trimmed)
		}
	}
	return mobiles
}

// AllMobiles returns all unique mobile numbers (from all receivers and default)
func (c *Config) AllMobiles() []string {
	mobileSet := make(map[string]struct{})

	for _, r := range c.Receivers {
		for _, p := range r.Mobiles {
			mobileSet[p] = struct{}{}
		}
	}

	for _, p := range c.DefaultReceiver.Mobiles {
		mobileSet[p] = struct{}{}
	}

	var mobiles []string
	for p := range mobileSet {
		mobiles = append(mobiles, p)
	}
	return mobiles
}
