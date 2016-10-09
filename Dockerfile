FROM cloudfoundry/cflinuxfs2
MAINTAINER Stephen Levine <stephen.levine@gmail.com>


ENV BUILDPACKS \
  http://github.com/cloudfoundry/java-buildpack \
  http://github.com/cloudfoundry/ruby-buildpack \
  http://github.com/cloudfoundry/nodejs-buildpack \
  http://github.com/cloudfoundry/go-buildpack \
  http://github.com/cloudfoundry/python-buildpack \
  http://github.com/cloudfoundry/php-buildpack \
  http://github.com/cloudfoundry/staticfile-buildpack \
  http://github.com/cloudfoundry/binary-buildpack

ENV \
  GO_VERSION=1.7 \
  DIEGO_VERSION=0.1482.0

RUN \
  curl -L "https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xz && \

RUN \
  mkdir -p /tmp/compile && \
  git -C /tmp/compile clone --single-branch https://github.com/cloudfoundry/diego-release && \
  cd /tmp/compile/diego-release && \
  git checkout "v${DIEGO_VERSION}" && \
  git submodule update --init --recursive \
    src/code.cloudfoundry.org/archiver \
    src/code.cloudfoundry.org/buildpackapplifecycle \
    src/code.cloudfoundry.org/bytefmt \
    src/code.cloudfoundry.org/cacheddownloader \
    src/github.com/cloudfoundry-incubator/candiedyaml \
    src/github.com/cloudfoundry/systemcerts

RUN \
  export PATH=/usr/local/go/bin:$PATH && \
  export GOPATH=/tmp/compile/diego-release && \
  go build -o /tmp/lifecycle/launcher code.cloudfoundry.org/buildpackapplifecycle/launcher && \
  go build -o /tmp/lifecycle/builder code.cloudfoundry.org/buildpackapplifecycle/builder

USER vcap

ENV \
  CF_INSTANCE_ADDR= \
  CF_INSTANCE_PORT= \
  CF_INSTANCE_PORTS=[] \
  CF_INSTANCE_IP=0.0.0.0 \
  CF_STACK=cflinuxfs2 \
  HOME=/home/vcap \
  MEMORY_LIMIT=512m \
  VCAP_SERVICES={}

ENV VCAP_APPLICATION '{ \
    "limits": {"fds": 16384, "mem": 512, "disk": 1024}, \
    "application_name": "local", "name": "local", "space_name": "local-space", \
    "application_uris": ["localhost"], "uris": ["localhost"], \
    "application_id": "01d31c12-d066-495e-aca2-8d3403165360", \
    "application_version": "2b860df9-a0a1-474c-b02f-5985f53ea0bb", \
    "version": "2b860df9-a0a1-474c-b02f-5985f53ea0bb", \
    "space_id": "18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1" \
  }'

COPY . /tmp/app

RUN \
  mkdir -p /home/vcap/tmp && \
  cd /home/vcap && \
  /tmp/lifecycle/builder -buildpackOrder "$(echo "$BUILDPACKS" | tr -s ' ' ,)"

EXPOSE 8080

RUN \
  tar -C /home/vcap -xzf /tmp/droplet && \
  chown -R vcap:vcap /home/vcap

ENV \
  CF_INSTANCE_INDEX=0 \
  CF_INSTANCE_ADDR=0.0.0.0:8080 \
  CF_INSTANCE_PORT=8080 \
  CF_INSTANCE_PORTS='[{"external":8080,"internal":8080}]' \
  CF_INSTANCE_GUID=999db41a-508b-46eb-74d8-6f9c06c006da \
  INSTANCE_GUID=999db41a-508b-46eb-74d8-6f9c06c006da \
  INSTANCE_INDEX=0 \
  PORT=8080 \
  TMPDIR=/home/vcap/tmp

ENV VCAP_APPLICATION '{ \
    "limits": {"fds": 16384, "mem": 512, "disk": 1024}, \
    "application_name": "local", "name": "local", "space_name": "local-space", \
    "application_uris": ["localhost"], "uris": ["localhost"], \
    "application_id": "01d31c12-d066-495e-aca2-8d3403165360", \
    "application_version": "2b860df9-a0a1-474c-b02f-5985f53ea0bb", \
    "version": "2b860df9-a0a1-474c-b02f-5985f53ea0bb", \
    "space_id": "18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1", \
    "instance_id": "999db41a-508b-46eb-74d8-6f9c06c006da", \
    "host": "0.0.0.0", "instance_index": 0, "port": 8080 \
  }'

CMD cd /home/vcap/app && /tmp/lifecycle/launcher /home/vcap/app "$(jq -r .start_command /home/vcap/staging_info.yml)" ''
