package app_test

const envFixture = `CF_INSTANCE_ADDR=
CF_INSTANCE_IP=0.0.0.0
CF_INSTANCE_PORT=
CF_INSTANCE_PORTS=[]
CF_STACK=cflinuxfs2
HOME=/home/vcap
HOSTNAME=cflocal
LANG=en_US.UTF-8
MEMORY_LIMIT=512m
no_proxy=*.local, 169.254/16
PATH=/usr/local/bin:/usr/bin:/bin
PWD=/home/vcap
SHLVL=1
USER=vcap
_=/usr/bin/env
VCAP_APPLICATION={"application_id":"01d31c12-d066-495e-aca2-8d3403165360","application_name":"some-app","application_uris":["localhost"],"application_version":"2b860df9-a0a1-474c-b02f-5985f53ea0bb","limits":{"disk":1024,"fds":16384,"mem":512},"name":"some-app","space_id":"18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1","space_name":"cflocal-space","uris":["localhost"],"version":"18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1"}
VCAP_SERVICES={}
`
