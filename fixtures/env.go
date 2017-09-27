package fixtures

import (
	"sort"
	"strings"
)

func RunningEnv(overrides ...string) []string {
	env := map[string]string{}
	merge(env, runningEnv)
	merge(env, shellEnv)
	env["DEPS_DIR"] = "/home/vcap/deps"
	env["HOME"] = "/home/vcap/app"
	env["PWD"] = "/home/vcap/app"
	mergeEnv(env, overrides)
	return toSlice(env)
}

func ProvidedRunningEnv(overrides ...string) []string {
	env := map[string]string{}
	merge(env, runningEnv)
	mergeEnv(env, overrides)
	return toSlice(env)
}

func StagingEnv(overrides ...string) []string {
	env := map[string]string{}
	merge(env, stagingEnv)
	merge(env, shellEnv)
	env["PWD"] = "/home/vcap"
	mergeEnv(env, overrides)
	return toSlice(env)
}

func ProvidedStagingEnv(overrides ...string) []string {
	env := map[string]string{}
	merge(env, stagingEnv)
	mergeEnv(env, overrides)
	return toSlice(env)
}

func merge(m, n map[string]string) {
	for k, v := range n {
		m[k] = v
	}
}

func mergeEnv(m map[string]string, env []string) {
	for _, e := range env {
		kv := strings.SplitN(e, "=", 2)
		m[kv[0]] = kv[1]
	}
}

func toSlice(m map[string]string) []string {
	var out []string
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

var runningEnv = map[string]string{
	"CF_INSTANCE_ADDR":     "0.0.0.0:8080",
	"CF_INSTANCE_GUID":     "999db41a-508b-46eb-74d8-6f9c06c006da",
	"CF_INSTANCE_INDEX":    "0",
	"CF_INSTANCE_IP":       "0.0.0.0",
	"CF_INSTANCE_PORT":     "8080",
	"CF_INSTANCE_PORTS":    `[{"external":8080,"internal":8080}]`,
	"INSTANCE_GUID":        "999db41a-508b-46eb-74d8-6f9c06c006da",
	"INSTANCE_INDEX":       "0",
	"LANG":                 "en_US.UTF-8",
	"MEMORY_LIMIT":         "512m",
	"PATH":                 "/usr/local/bin:/usr/bin:/bin",
	"PORT":                 "8080",
	"TEST_ENV_KEY":         "test-env-value",
	"TEST_RUNNING_ENV_KEY": "test-running-env-value",
	"TMPDIR":               "/home/vcap/tmp",
	"USER":                 "vcap",
	"VCAP_APPLICATION":     `{"application_id":"01d31c12-d066-495e-aca2-8d3403165360","application_name":"some-app","application_uris":["localhost"],"application_version":"2b860df9-a0a1-474c-b02f-5985f53ea0bb","host":"0.0.0.0","instance_id":"999db41a-508b-46eb-74d8-6f9c06c006da","instance_index":0,"limits":{"disk":1024,"fds":16384,"mem":512},"name":"some-app","port":8080,"space_id":"18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1","space_name":"cflocal-space","uris":["localhost"],"version":"18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1"}`,
	"VCAP_SERVICES":        `{"some-type":[{"name":"some-name","label":"","tags":null,"plan":"","credentials":null,"syslog_drain_url":null,"provider":null,"volume_mounts":null}]}`,
}

var stagingEnv = map[string]string{
	"CF_INSTANCE_ADDR":     "",
	"CF_INSTANCE_IP":       "0.0.0.0",
	"CF_INSTANCE_PORT":     "",
	"CF_INSTANCE_PORTS":    "[]",
	"CF_STACK":             "cflinuxfs2",
	"HOME":                 "/home/vcap",
	"LANG":                 "en_US.UTF-8",
	"MEMORY_LIMIT":         "1024m",
	"PATH":                 "/usr/local/bin:/usr/bin:/bin",
	"TEST_ENV_KEY":         "test-env-value",
	"TEST_STAGING_ENV_KEY": "test-staging-env-value",
	"USER":                 "vcap",
	"VCAP_APPLICATION":     `{"application_id":"01d31c12-d066-495e-aca2-8d3403165360","application_name":"some-app","application_uris":["localhost"],"application_version":"2b860df9-a0a1-474c-b02f-5985f53ea0bb","limits":{"disk":4096,"fds":16384,"mem":1024},"name":"some-app","space_id":"18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1","space_name":"cflocal-space","uris":["localhost"],"version":"18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1"}`,
	"VCAP_SERVICES":        `{"some-type":[{"name":"some-name","label":"","tags":null,"plan":"","credentials":null,"syslog_drain_url":null,"provider":null,"volume_mounts":null}]}`,
}

var shellEnv = map[string]string{
	"_":        "/usr/bin/env",
	"SHLVL":    "1",
	"HOSTNAME": "cflocal",
}
