package local

import (
	"io/ioutil"
	"os"

	"github.com/sclevine/cflocal/service"
	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	Path string
}

func (c *Config) Load() (*LocalYML, error) {
	localYML := &LocalYML{}
	yamlBytes, err := ioutil.ReadFile(c.Path)
	if pathErr, ok := err.(*os.PathError); ok && pathErr.Op == "open" {
		return localYML, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(yamlBytes, localYML); err != nil {
		return nil, err
	}
	return localYML, nil

}

func (c *Config) Save(localYML *LocalYML) error {
	yamlBytes, err := yaml.Marshal(localYML)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.Path, yamlBytes, 0666)
}

type LocalYML struct {
	Applications []*AppConfig `yaml:"applications,omitempty"`
}

type AppConfig struct {
	Name       string            `yaml:"name"`
	Command    string            `yaml:"command,omitempty"`
	DiskQuota  string            `yaml:"disk_quota,omitempty"`
	Memory     string            `yaml:"memory,omitempty"`
	StagingEnv map[string]string `yaml:"staging_env,omitempty"`
	RunningEnv map[string]string `yaml:"running_env,omitempty"`
	Env        map[string]string `yaml:"env,omitempty"`
	Services   service.Services  `yaml:"services,omitempty"`
}

type NetworkConfig struct {
	ContainerID string
	HostIP      string
	HostPort    string
}
