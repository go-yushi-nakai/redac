package redac

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/adrg/xdg"
)

type ConfigFile struct {
	Contexts map[string]*ConfigContext `json:"contexts"`
}

type ConfigContext struct {
	Name         string `json:"name"`
	Endpoint     string `json:"endpoint"`
	APIKey       string `json:"apiKey"`
	DataSourceID int    `json:"dataSourceID"`
}

var configFile = "redac/config.json"

func AddConfigContext(name, endpoint, apiKey string, dsID int) error {
	cf, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cc := ConfigContext{
		Name:         name,
		Endpoint:     endpoint,
		APIKey:       apiKey,
		DataSourceID: dsID,
	}
	if _, ok := cf.Contexts[cc.Name]; ok {
		return fmt.Errorf("context %s already exists", cc.Name)
	}
	cf.Contexts[cc.Name] = &cc

	if err := SaveConfig(cf); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

func DeleteConfigContext(name string) error {
	cf, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if _, ok := cf.Contexts[name]; !ok {
		return fmt.Errorf("context %s does not exist", name)
	}
	delete(cf.Contexts, name)

	if err := SaveConfig(cf); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

func LoadConfig() (*ConfigFile, error) {
	configFilePath, err := xdg.ConfigFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	cf := ConfigFile{
		Contexts: make(map[string]*ConfigContext),
	}
	b, err := os.ReadFile(configFilePath)
	if err != nil {
		return &cf, nil
	}
	if err := json.Unmarshal(b, &cf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	if cf.Contexts == nil {
		cf.Contexts = make(map[string]*ConfigContext)
	}
	return &cf, nil
}

func SaveConfig(c *ConfigFile) error {
	configFilePath, err := xdg.ConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	b, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFilePath, b, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if err := os.Chmod(configFilePath, 0600); err != nil {
		return fmt.Errorf("failed to set permission on config file: %w", err)
	}
	return nil
}
