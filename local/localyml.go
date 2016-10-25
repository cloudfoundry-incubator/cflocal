package local

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Path string
}

func (c *Config) Load() (*LocalYML, error) {
	yamlBytes, err := ioutil.ReadFile(c.Path)
	if err != nil {
		return nil, err
	}
	localYML := &LocalYML{}
	if err := yaml.Unmarshal(yamlBytes, localYML); err != nil {
		return nil, err
	}
	return localYML, nil

}

func (c *Config) Set(localYML *LocalYML) error {
	yamlBytes, err := yaml.Marshal(localYML)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.Path, yamlBytes, 0644)
}

type LocalYML struct {
	Applications []*AppConfig `yaml:"applications,omitempty"`
}

type AppConfig struct {
	Name       string            `yaml:"name"`
	Buildpack  string            `yaml:"buildpack,omitempty"`
	Droplet    string            `yaml:"droplet,omitempty"`
	Command    string            `yaml:"command,omitempty"`
	StagingEnv map[string]string `yaml:"staging_env,omitempty"`
	RunningEnv map[string]string `yaml:"running_env,omitempty"`
	Env        map[string]string `yaml:"env,omitempty"`
}
