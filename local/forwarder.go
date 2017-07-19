package local

import (
	"bytes"
	"fmt"
	"io"
	"text/template"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"

	"github.com/sclevine/cflocal/engine"
	"github.com/sclevine/cflocal/service"
	"io/ioutil"
)

const ForwardScript = `
	set -e
	{{if .Forwards -}}
	echo 'Forwarding:{{range .Forwards}} {{.Name}}{{end}}'
	sshpass -f /tmp/ssh-code ssh -N \
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

func (f *Forwarder) Forward(config *ForwardConfig) (health <-chan string, err error) {
	containerConfig, err := f.buildContainerConfig(config.ForwardConfig)
	if err != nil {
		return nil, err
	}
	contr, err := f.Engine.NewContainer(containerConfig, nil)
	if err != nil {
		return nil, err
	}
	if err := contr.CopyTo(config.SSHPass, "/usr/bin/sshpass"); err != nil {
		return nil, err
	}
	health, done := contr.HealthCheck()
	go func() {
		defer contr.Close()
		defer close(done)
		prefix := config.Color("[%s tunnel] ", config.AppName)
		for {
			select {
			case <-f.Exit:
				return
			default:
				code, err := config.ForwardConfig.Code()
				if err != nil {
					fmt.Fprintf(f.Logs, "%sError: %s", prefix, err)
					break
				}
				codeStream := engine.NewStream(ioutil.NopCloser(bytes.NewBufferString(code)), int64(len(code)))
				if err := contr.CopyTo(codeStream, "/tmp/ssh-code"); err != nil {
					fmt.Fprintf(f.Logs, "%sError: %s", prefix, err)
					break
				}
				status, err := contr.Start(prefix, f.Logs, nil)
				if err != nil {
					fmt.Fprintf(f.Logs, "%sError: %s", prefix, err)
					break
				}
				fmt.Fprintf(f.Logs, "%sExited with status: %d", prefix, status)
			}
		}
	}()
	return health, nil
}

func (f *Forwarder) buildContainerConfig(forwardConfig *service.ForwardConfig) (*container.Config, error) {
	scriptBuf := &bytes.Buffer{}
	tmpl := template.Must(template.New("").Parse(ForwardScript))
	if err := tmpl.Execute(scriptBuf, forwardConfig); err != nil {
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
