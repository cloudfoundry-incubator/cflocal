package local

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
    src/github.com/cloudfoundry-incubator/candiedyaml \
    src/github.com/cloudfoundry/systemcerts && \
  export PATH=/usr/local/go/bin:$PATH && \
  export GOPATH=/tmp/diego-release && \
  go build -o /tmp/lifecycle/launcher code.cloudfoundry.org/buildpackapplifecycle/launcher && \
  go build -o /tmp/lifecycle/builder code.cloudfoundry.org/buildpackapplifecycle/builder && \
  rm -rf /tmp/diego-release /usr/local/go

USER vcap

RUN mkdir -p /tmp/app /home/vcap/tmp
`
