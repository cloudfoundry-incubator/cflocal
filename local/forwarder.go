package local

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"text/template"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"

	"github.com/sclevine/cflocal/engine"
	"github.com/sclevine/cflocal/service"
)

const ForwardScript = `
	set -e
	{{if .Forwards -}}
	echo 'Forwarding:{{range .Forwards}} {{.Name}}{{end}}'
	sshpass -p '{{.Code}}' ssh -N \
	    -o PermitLocalCommand=yes -o LocalCommand="touch /tmp/healthy" \
		-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no \
		-o LogLevel=ERROR -o ExitOnForwardFailure=yes \
		-o ServerAliveInterval=10 -o ServerAliveCountMax=60 \
		-p '{{.Port}}' '{{.User}}@{{.Host}}' \
		{{- range $i, $_ := .Forwards}}
		{{- if $i}} \{{end}}
		-L '{{.From}}:{{.To}}'
		{{- end}}
	{{end -}}
`

type Forwarder struct {
	StackVersion string
	Logs         io.Writer
	Exit         <-chan struct{}
	Engine       Engine
}

type ForwardConfig struct {
	AppName       string
	SSHPass       engine.Stream
	Color         Colorizer
	ForwardConfig *service.ForwardConfig
}

func (f *Forwarder) Run(config *ForwardConfig) (health <-chan string, err error) {
	sshpassBuf := &bytes.Buffer{}
	if err := config.SSHPass.Out(sshpassBuf); err != nil {
		return nil, err
	}

	prefix := config.Color("[%s tunnel] ", config.AppName)
	sshHealth := make(chan string)
	go func() {
		for {
			select {
			case <-f.Exit:
				return
			default:
				if err := f.start(sshHealth, config.ForwardConfig, prefix, sshpassBuf.Bytes()); err != nil {
					fmt.Fprintf(f.Logs, "%s%s", prefix, err)
				}
			}
		}
	}()
	return sshHealth, nil
}

func (f *Forwarder) start(health chan<- string, config *service.ForwardConfig, prefix string, sshpass []byte) error {
	containerConfig, err := f.buildContainerConfig(config)
	if err != nil {
		return err
	}
	contr, err := f.Engine.NewContainer(containerConfig, nil)
	if err != nil {
		return err
	}
	defer contr.Close()

	sshpassStream := engine.NewStream(ioutil.NopCloser(bytes.NewBuffer(sshpass)), int64(len(sshpass)))
	if err := contr.CopyTo(sshpassStream, "/usr/bin/sshpass"); err != nil {
		return err
	}
	defer close(contr.HealthCheck(health))
	_, err = contr.Start(prefix, f.Logs, nil)
	return err
}

func (f *Forwarder) buildContainerConfig(forwardConfig *service.ForwardConfig) (*container.Config, error) {
	code, err := forwardConfig.Code()
	if err != nil {
		return nil, err
	}

	options := struct {
		*service.ForwardConfig
		Code string
	}{forwardConfig, code}
	scriptBuf := &bytes.Buffer{}
	tmpl := template.Must(template.New("").Parse(ForwardScript))
	if err := tmpl.Execute(scriptBuf, options); err != nil {
		return nil, err
	}
	ports := nat.PortSet{}
	for _, f := range forwardConfig.Forwards {
		ports[nat.Port(fmt.Sprintf("%s/tcp", f.From))] = struct{}{}
	}

	return &container.Config{
		Hostname:     "cflocal",
		User:         "vcap",
		ExposedPorts: ports,
		Healthcheck: &container.HealthConfig{
			Test: []string{"CMD", "test", "-f", "/tmp/healthy"},
		},
		Image: "cloudfoundry/cflinuxfs2:" + f.StackVersion,
		Entrypoint: strslice.StrSlice{
			"/bin/bash", "-c", scriptBuf.String(),
		},
	}, nil
}
