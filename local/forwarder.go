package local

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"text/template"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"

	"github.com/docker/go-connections/nat"
	"github.com/sclevine/cflocal/engine"
	"github.com/sclevine/cflocal/local/outlock"
	"github.com/sclevine/cflocal/service"
)

const ForwardScript = `
	{{if .Forwards -}}
	echo 'Forwarding:{{range .Forwards}} {{.Name}}{{end}}'
	sshpass -f /tmp/ssh-code ssh -4 -N \
	    -o PermitLocalCommand=yes -o LocalCommand="touch /tmp/healthy" \
		-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no \
		-o LogLevel=ERROR -o ExitOnForwardFailure=yes \
		-o ServerAliveInterval=10 -o ServerAliveCountMax=60 \
		-p '{{.Port}}' '{{.User}}@{{.Host}}' \
		{{- range $i, $_ := .Forwards}}
		{{- if $i}} \{{end}}
		-L '{{.From}}:{{.To}}'
		{{- end}}
	rm -f /tmp/healthy
	{{- end}}
`

type Forwarder struct {
	StackVersion string
	Logs         io.Writer
	Exit         <-chan struct{}
	Engine       Engine
}

type ForwardConfig struct {
	AppName          string
	SSHPass          engine.Stream
	Color            Colorizer
	ForwardConfig    *service.ForwardConfig
	HostIP, HostPort string
	Wait             <-chan time.Time
}

func (f *Forwarder) Forward(config *ForwardConfig) (health <-chan string, done func(), id string, err error) {
	output := outlock.New(f.Logs)

	netHostConfig := &container.HostConfig{PortBindings: nat.PortMap{
		"8080/tcp": {{HostIP: config.HostIP, HostPort: config.HostPort}},
	}}
	netContr, err := f.Engine.NewContainer(f.buildNetContainerConfig(), netHostConfig)
	if err != nil {
		return nil, nil, "", err
	}
	// TODO: wait for network container to fully start in Background
	if err := netContr.Background(); err != nil {
		return nil, nil, "", err
	}

	networkMode := "container:" + netContr.ID()
	containerConfig, err := f.buildContainerConfig(config.ForwardConfig)
	if err != nil {
		return nil, nil, "", err
	}
	hostConfig := &container.HostConfig{NetworkMode: container.NetworkMode(networkMode)}
	contr, err := f.Engine.NewContainer(containerConfig, hostConfig)
	if err != nil {
		return nil, nil, "", err
	}

	if err := contr.CopyTo(config.SSHPass, "/usr/bin/sshpass"); err != nil {
		return nil, nil, "", err
	}

	prefix := config.Color("[%s tunnel] ", config.AppName)
	go func() {
		for {
			select {
			case <-f.Exit: // TODO: refactor shutdown to remove
				return
			case <-config.Wait:
				code, err := config.ForwardConfig.Code()
				if err != nil {
					fmt.Fprintf(output, "%sError: %s\n", prefix, err)
					continue
				}
				codeStream := engine.NewStream(ioutil.NopCloser(bytes.NewBufferString(code)), int64(len(code)))
				if err := contr.CopyTo(codeStream, "/tmp/ssh-code"); err != nil {
					fmt.Fprintf(output, "%sError: %s\n", prefix, err)
					continue
				}
				status, err := contr.Start(prefix, output, nil)
				if err != nil {
					fmt.Fprintf(output, "%sError: %s\n", prefix, err)
					continue
				}
				fmt.Fprintf(output, "%sExited with status: %d\n", prefix, status)
			}
		}
	}()
	done = func() {
		defer netContr.Close()
		defer contr.Close()
		output.Disable()
	}
	return contr.HealthCheck(), done, netContr.ID(), nil
}

func (f *Forwarder) buildContainerConfig(forwardConfig *service.ForwardConfig) (*container.Config, error) {
	scriptBuf := &bytes.Buffer{}
	tmpl := template.Must(template.New("").Parse(ForwardScript))
	if err := tmpl.Execute(scriptBuf, forwardConfig); err != nil {
		return nil, err
	}

	return &container.Config{
		User: "vcap",
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD", "test", "-f", "/tmp/healthy"},
			Interval: time.Second,
			Retries:  30,
		},
		Image: "cloudfoundry/cflinuxfs2:" + f.StackVersion,
		Entrypoint: strslice.StrSlice{
			"/bin/bash", "-c", scriptBuf.String(),
		},
	}, nil
}

func (f *Forwarder) buildNetContainerConfig() *container.Config {
	return &container.Config{
		Hostname:     "cflocal",
		User:         "vcap",
		ExposedPorts: nat.PortSet{"8080/tcp": {}},
		Image:        "cloudfoundry/cflinuxfs2:" + f.StackVersion,
		Entrypoint: strslice.StrSlice{
			"tail", "-f", "/dev/null",
		},
	}
}
