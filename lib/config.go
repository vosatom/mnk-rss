package lib

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BaseURL           string                `yaml:"baseUrl"`
	OwsURL            string                `yaml:"owsUrl"`
	ProjectURL        string                `yaml:"projectUrl"`
	DefaultProjection string                `yaml:"defaultProjection"`
	DefaultExtent     []float32             `yaml:"defaultExtent"`
	Paths             map[string]FeedConfig `yaml:"paths"`
	Bookmarks         struct {
		Group       string `yaml:"group"`
		DefaultCity string `yaml:"defaultCity"`
	} `yaml:"bookmarks"`
}

type FeedConfig struct {
	Type        string                 `yaml:"type"`
	Title       string                 `yaml:"title"`
	Description string                 `yaml:"description"`
	Language    string                 `yaml:"language"`
	Options     map[string]interface{} `yaml:"options"`
	Params      map[string]interface{} `yaml:"params"`
}

func ReadConfig(configPath string) (Config, error) {
	var config Config
	file, err := os.Open(configPath)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	if file != nil {
		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(&config); err != nil {
			return Config{}, err
		}
	}

	return config, nil
}
