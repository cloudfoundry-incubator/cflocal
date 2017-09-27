package local

import (
	"github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"

	"github.com/sclevine/cflocal/engine"
)

const dockerfile = `
FROM cloudfoundry/cflinuxfs2:{{.StackVersion}}
MAINTAINER CF Local <cflocal@sclevine.org>

RUN \
  apt-get update && \
  apt-get -y install sshpass && \
  apt-get clean

RUN \
  curl -L "https://storage.googleapis.com/golang/go{{.GoVersion}}.linux-amd64.tar.gz" | tar -C /usr/local -xz && \
  git -C /tmp clone --single-branch https://github.com/cloudfoundry/diego-release && \
  cd /tmp/diego-release && \
  git checkout "v{{.DiegoVersion}}" && \
  git submodule update --init --recursive \
    src/code.cloudfoundry.org/archiver \
    src/code.cloudfoundry.org/buildpackapplifecycle \
    src/code.cloudfoundry.org/bytefmt \
    src/code.cloudfoundry.org/cacheddownloader \
    src/code.cloudfoundry.org/lager \
    src/code.cloudfoundry.org/systemcerts \
    src/gopkg.in/yaml.v2 && \
  export PATH=/usr/local/go/bin:$PATH && \
  export GOPATH=/tmp/diego-release && \
  go build -o /tmp/lifecycle/launcher code.cloudfoundry.org/buildpackapplifecycle/launcher && \
  go build -o /tmp/lifecycle/builder code.cloudfoundry.org/buildpackapplifecycle/builder && \
  rm -rf /tmp/diego-release /usr/local/go

USER vcap

RUN mkdir -p /tmp/app /tmp/cache /home/vcap/tmp /tmp/buildpacks

{{range .Buildpacks}}
RUN curl -L -o /tmp/buildpack.zip "{{.URL}}" && unzip /tmp/buildpack.zip -d /tmp/buildpacks/{{.MD5}} && rm /tmp/buildpack.zip
{{end}}
`

type DockerEngine struct {
	Docker *docker.Client
	Exit   <-chan struct{}
}

func (d *DockerEngine) NewContainer(config *container.Config, hostConfig *container.HostConfig) (Container, error) {
	contr, err := engine.NewContainer(d.Docker, config, hostConfig)
	if err != nil {
		return nil, err
	}
	contr.Exit = d.Exit
	return contr, nil
}
