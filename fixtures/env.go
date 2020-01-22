package fixtures

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	ipv4Regexp = `(:?[0-9]{1,3}\.){3}[0-9]{1,3}`
	guidRegexp = `[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}`
)

func RunningEnv(name string, mem int64, service string, overrides ...string) []string {
	env := map[string]string{}
	merge(env, runningEnv(name, mem, service))
	merge(env, shellEnv)
	env["DEPS_DIR"] = "/home/vcap/deps"
	env["HOME"] = "/home/vcap/app"
	env["PWD"] = "/home/vcap/app"
	mergeEnv(env, overrides)
	return toSlice(env)
}

func StagingEnv(name string, mem int64, service string, overrides ...string) []string {
	env := map[string]string{}
	merge(env, stagingEnv(name, mem, service))
	merge(env, shellEnv)
	env["PWD"] = "/tmp/app"
	mergeEnv(env, overrides)
	return toSlice(env)
}

func runningEnv(name string, mem int64, service string) map[string]string {
	return map[string]string{
		"CF_INSTANCE_ADDR":        ipv4Regexp + ":8080",
		"CF_INSTANCE_GUID":        guidRegexp,
		"CF_INSTANCE_INDEX":       "[0-9]+",
		"CF_INSTANCE_INTERNAL_IP": ipv4Regexp,
		"CF_INSTANCE_IP":          ipv4Regexp,
		"CF_INSTANCE_PORT":        "8080",
		"CF_INSTANCE_PORTS":       qm(`[{"external":8080,"internal":8080}]`),
		"INSTANCE_GUID":           guidRegexp,
		"INSTANCE_INDEX":          "[0-9]+",
		"LANG":                    qm("en_US.UTF-8"),
		"MEMORY_LIMIT":            fmt.Sprintf("%dm", mem),
		"PACK_APP_DISK":           "[0-9]+",
		"PACK_APP_MEM":            fmt.Sprintf("%d", mem),
		"PACK_APP_NAME":           name,
		"PATH":                    "/usr/local/bin:/usr/bin:/bin",
		"PORT":                    "8080",
		"TMPDIR":                  "/home/vcap/tmp",
		"USER":                    "vcap",
		"VCAP_APP_HOST":           "0.0.0.0",
		"VCAP_APPLICATION":        vcapAppRunning(name, mem),
		"VCAP_APP_PORT":           "8080",
		"VCAP_SERVICES":           vcapServices(service),
	}
}

func stagingEnv(name string, mem int64, service string) map[string]string {
	return map[string]string{
		"CF_INSTANCE_ADDR":        "",
		"CF_INSTANCE_INTERNAL_IP": ipv4Regexp,
		"CF_INSTANCE_IP":          ipv4Regexp,
		"CF_INSTANCE_PORT":        "",
		"CF_INSTANCE_PORTS":       qm("[]"),
		"CF_STACK":                "cflinuxfs3",
		"HOME":                    "/home/vcap",
		"LANG":                    qm("en_US.UTF-8"),
		"MEMORY_LIMIT":            fmt.Sprintf("%dm", mem),
		"PACK_APP_DISK":           "[0-9]+",
		"PACK_APP_MEM":            fmt.Sprintf("%d", mem),
		"PACK_APP_NAME":           name,
		"PATH":                    "/usr/local/bin:/usr/bin:/bin",
		"USER":                    "vcap",
		"VCAP_APPLICATION":        vcapAppStaging(name, mem),
		"VCAP_SERVICES":           vcapServices(service),
	}
}

var shellEnv = map[string]string{
	"_":        "/usr/bin/env",
	"SHLVL":    "1",
	"HOSTNAME": "some-name",
}

func vcapAppRunning(name string, mem int64) string {
	return qm(`{`) +
		qm(`"application_id":"`) + guidRegexp + qm(`",`) +
		qm(`"application_name":"`) + name + qm(`",`) +
		qm(`"application_uris":["`) + name + qm(`.local"],`) +
		qm(`"application_version":"`) + guidRegexp + qm(`",`) +
		qm(`"host":"0.0.0.0",`) +
		qm(`"instance_id":"`) + guidRegexp + qm(`",`) +
		qm(`"instance_index":`) + "[0-9]+" + qm(`,`) +
		qm(`"limits":{`) + fmt.Sprintf(`"disk":[0-9]+,"fds":[0-9]+,"mem":%d`, mem) + qm("},") +
		qm(`"name":"`) + name + qm(`",`) +
		qm(`"port":8080,`) +
		qm(`"space_id":"`) + guidRegexp + qm(`",`) +
		qm(`"space_name":"`) + name + qm(`-space",`) +
		qm(`"uris":["`) + name + qm(`.local"],`) +
		qm(`"version":"`) + guidRegexp + qm(`"`) +
		qm(`}`)
}

func vcapAppStaging(name string, mem int64) string {
	return qm(`{`) +
		qm(`"application_id":"`) + guidRegexp + qm(`",`) +
		qm(`"application_name":"`) + name + qm(`",`) +
		qm(`"application_uris":["`) + name + qm(`.local"],`) +
		qm(`"application_version":"`) + guidRegexp + qm(`",`) +
		qm(`"limits":{`) + fmt.Sprintf(`"disk":[0-9]+,"fds":[0-9]+,"mem":%d`, mem) + qm("},") +
		qm(`"name":"`) + name + qm(`",`) +
		qm(`"space_id":"`) + guidRegexp + qm(`",`) +
		qm(`"space_name":"`) + name + qm(`-space",`) +
		qm(`"uris":["`) + name + qm(`.local"],`) +
		qm(`"version":"`) + guidRegexp + qm(`"`) +
		qm(`}`)
}

func vcapServices(service string) string {
	if service == "" {
		return `{}`
	}
	return qm(`{"`) + service + qm(`":[`) + `.+` + qm(`]}`)
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
		out = append(out, anchor(qm(k+"=")+v))
	}
	sort.Strings(out)
	return out
}

func qm(s string) string {
	return regexp.QuoteMeta(s)
}

func anchor(re string) string {
	return fmt.Sprintf("^%s$", re)
}
