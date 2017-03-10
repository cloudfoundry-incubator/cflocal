package local

const dockerfile = `
FROM cloudfoundry/cflinuxfs2:{{.StackVersion}}
MAINTAINER CF Local <cflocal@sclevine.org>

RUN \
  apt-get update && \
  apt-get -y install iptables python-setuptools && \
  apt-get clean

RUN \
  git -C /tmp clone --single-branch https://github.com/sclevine/sshuttle && \
  cd /tmp/sshuttle && \
  git checkout "v{{.ShuttleVersion}}" && \
  ./setup.py install && \
  rm -rf /tmp/sshuttle

RUN \
  curl -L "https://storage.googleapis.com/golang/go{{.GoVersion}}.linux-amd64.tar.gz" | tar -C /usr/local -xz

RUN \
  mkdir -p /tmp/compile && \
  git -C /tmp/compile clone --single-branch https://github.com/cloudfoundry/diego-release && \
  cd /tmp/compile/diego-release && \
  git checkout "v{{.DiegoVersion}}" && \
  git submodule update --init --recursive \
    src/code.cloudfoundry.org/archiver \
    src/code.cloudfoundry.org/buildpackapplifecycle \
    src/code.cloudfoundry.org/bytefmt \
    src/code.cloudfoundry.org/cacheddownloader \
    src/github.com/cloudfoundry-incubator/candiedyaml \
    src/github.com/cloudfoundry/systemcerts && \
  export PATH=/usr/local/go/bin:$PATH && \
  export GOPATH=/tmp/compile/diego-release && \
  go build -o /tmp/lifecycle/launcher code.cloudfoundry.org/buildpackapplifecycle/launcher && \
  go build -o /tmp/lifecycle/builder code.cloudfoundry.org/buildpackapplifecycle/builder && \
  rm -rf /tmp/compile

USER vcap

RUN mkdir -p /tmp/app /home/vcap/tmp
`
