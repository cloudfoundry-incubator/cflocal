package fixtures

import "fmt"

func RunRSyncScript() string {
	return fmt.Sprintf(runnerScript, "\n\trsync -a /tmp/local/ /home/vcap/app/", rsyncRunningToLocal)
}

func CommitScript() string {
	return fmt.Sprintf(runnerScript, "", "")
}

func StageRSyncScript() string {
	return fmt.Sprintf(stageScript, "", "\n\trsync -a /tmp/app/ /tmp/local/")
}

func ForwardScript() string {
	return forwardScript
}

const forwardScript = `
	echo 'Forwarding: some-name some-other-name'
	sshpass -f /tmp/ssh-code ssh -4 -N \
	    -o PermitLocalCommand=yes -o LocalCommand="touch /tmp/healthy" \
		-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no \
		-o LogLevel=ERROR -o ExitOnForwardFailure=yes \
		-o ServerAliveInterval=10 -o ServerAliveCountMax=60 \
		-p 'some-port' 'some-user@some-ssh-host' \
		-L 'some-from:some-to' \
		-L 'some-other-from:some-other-to'
	rm -f /tmp/healthy
`

const rsyncRunningToLocal = `
	if [[ -z $(ls -A /tmp/local) ]]; then
		rsync -a /home/vcap/app/ /tmp/local/
	fi`

const runnerScript = `
	set -e%s
	if [[ ! -z $(ls -A /home/vcap/app) ]]; then
		exclude='--exclude=./app'
	fi
	tar $exclude -C /home/vcap -xzf /tmp/droplet
	chown -R vcap:vcap /home/vcap%s
	command=$1
	if [[ -z $command ]]; then
		command=$(jq -r .start_command /home/vcap/staging_info.yml)
	fi
	exec /tmp/lifecycle/launcher /home/vcap/app "$command" ''
`

const stageScript = `
	set -e
	su vcap -c "unzip -qq /tmp/some-checksum-one.zip -d /tmp/buildpacks/some-checksum-one" && rm /tmp/some-checksum-one.zip
	su vcap -c "unzip -qq /tmp/some-checksum-two.zip -d /tmp/buildpacks/some-checksum-two" && rm /tmp/some-checksum-two.zip

	chown -R vcap:vcap /tmp/app /tmp/cache
	%ssu vcap -p -c "PATH=$PATH exec /tmp/lifecycle/builder -buildpackOrder $0 -skipDetect=$1"%s
`
