package service

type Service struct {
	Name           string            `json:"name" yaml:"name"`
	Label          string            `json:"label" yaml:"label"`
	Tags           []string          `json:"tags" yaml:"tags"`
	Plan           string            `json:"plan" yaml:"plan"`
	Credentials    map[string]string `json:"credentials" yaml:"credentials"`
	SyslogDrainURL *string           `json:"syslog_drain_url" yaml:"syslog_drain_url,omitempty"`
	Provider       *string           `json:"provider" yaml:"provider,omitempty"`
	VolumeMounts   []string          `json:"volume_mounts" yaml:"volume_mounts,omitempty"`
}

type Services map[string][]Service

type ForwardConfig struct {
	Host     string
	Port     string
	User     string
	Code     string
	Forwards []Forward
}

type Forward struct {
	Name string
	From string
	To   string
}
