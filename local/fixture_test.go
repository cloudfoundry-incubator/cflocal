package local_test

const runScriptFixture = `
	set -e
	echo 'Forwarding: some-name some-other-name'
	sshpass -p 'some-code' ssh -f -N \
		-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no \
		-o LogLevel=ERROR -o ExitOnForwardFailure=yes \
		-o ServerAliveInterval=10 -o ServerAliveCountMax=60 \
		-p 'some-port' 'some-user@some-ssh-host' \
		-L 'some-from:some-to' \
		-L 'some-other-from:some-other-to'
	tar --exclude=./app -C /home/vcap -xzf /tmp/droplet
	chown -R vcap:vcap /home/vcap
	command=$1
	if [[ -z $command ]]; then
		command=$(jq -r .start_command /home/vcap/staging_info.yml)
	fi
	exec /tmp/lifecycle/launcher /home/vcap/app "$command" ''
`

const commitScriptFixture = `
	set -e
	tar --exclude= -C /home/vcap -xzf /tmp/droplet
	chown -R vcap:vcap /home/vcap
	command=$1
	if [[ -z $command ]]; then
		command=$(jq -r .start_command /home/vcap/staging_info.yml)
	fi
	exec /tmp/lifecycle/launcher /home/vcap/app "$command" ''
`

var runnerEnvFixture = []string{
	"CF_INSTANCE_ADDR=0.0.0.0:8080",
	"CF_INSTANCE_GUID=999db41a-508b-46eb-74d8-6f9c06c006da",
	"CF_INSTANCE_INDEX=0",
	"CF_INSTANCE_IP=0.0.0.0",
	"CF_INSTANCE_PORT=8080",
	`CF_INSTANCE_PORTS=[{"external":8080,"internal":8080}]`,
	"INSTANCE_GUID=999db41a-508b-46eb-74d8-6f9c06c006da",
	"INSTANCE_INDEX=0",
	"LANG=en_US.UTF-8",
	"MEMORY_LIMIT=1024m",
	"PATH=/usr/local/bin:/usr/bin:/bin",
	"PORT=8080",
	"TEST_ENV_KEY=test-env-value",
	"TEST_RUNNING_ENV_KEY=test-running-env-value",
	"TMPDIR=/home/vcap/tmp",
	"USER=vcap",
	`VCAP_APPLICATION={"application_id":"01d31c12-d066-495e-aca2-8d3403165360","application_name":"some-app","application_uris":["localhost"],"application_version":"2b860df9-a0a1-474c-b02f-5985f53ea0bb","host":"0.0.0.0","instance_id":"999db41a-508b-46eb-74d8-6f9c06c006da","instance_index":0,"limits":{"disk":1024,"fds":16384,"mem":512},"name":"some-app","port":8080,"space_id":"18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1","space_name":"cflocal-space","uris":["localhost"],"version":"18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1"}`,
	`VCAP_SERVICES={"some-type":[{"name":"some-name","label":"","tags":null,"plan":"","credentials":null,"syslog_drain_url":null,"provider":null,"volume_mounts":null}]}`,
}

var stagerEnvFixture = []string{
	"CF_INSTANCE_ADDR=",
	"CF_INSTANCE_IP=0.0.0.0",
	"CF_INSTANCE_PORT=",
	"CF_INSTANCE_PORTS=[]",
	"CF_STACK=cflinuxfs2",
	"HOME=/home/vcap",
	"LANG=en_US.UTF-8",
	"MEMORY_LIMIT=1024m",
	"PATH=/usr/local/bin:/usr/bin:/bin",
	"TEST_ENV_KEY=test-env-value",
	"TEST_STAGING_ENV_KEY=test-staging-env-value",
	"USER=vcap",
	`VCAP_APPLICATION={"application_id":"01d31c12-d066-495e-aca2-8d3403165360","application_name":"some-app","application_uris":["localhost"],"application_version":"2b860df9-a0a1-474c-b02f-5985f53ea0bb","limits":{"disk":1024,"fds":16384,"mem":512},"name":"some-app","space_id":"18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1","space_name":"cflocal-space","uris":["localhost"],"version":"18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1"}`,
	`VCAP_SERVICES={"some-type":[{"name":"some-name","label":"","tags":null,"plan":"","credentials":null,"syslog_drain_url":null,"provider":null,"volume_mounts":null}]}`,
}
