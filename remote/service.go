package remote

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/sclevine/cflocal/service"
)

const firstForwardedServicePort uint = 40000

func (a *App) Services(name string) (service.Services, error) {
	appEnvJSON, _, err := a.get(name, "/env")
	if err != nil {
		return nil, err
	}
	defer appEnvJSON.Close()
	var env struct {
		SystemEnvJSON struct {
			VCAPServices service.Services `json:"VCAP_SERVICES"`
		} `json:"system_env_json"`
	}
	if err := json.NewDecoder(appEnvJSON).Decode(&env); err != nil {
		return nil, err
	}
	return env.SystemEnvJSON.VCAPServices, nil
}

func (a *App) Forward(name string, svcs service.Services) (service.Services, *service.ForwardConfig, error) {
	var err error
	config := &service.ForwardConfig{}

	if config.Host, config.Port, err = a.sshEndpoint(); err != nil {
		return nil, nil, err
	}

	if err := a.checkAuth(); err != nil {
		return nil, nil, err
	}
	appModel, err := a.CLI.GetApp(name)
	if err != nil {
		return nil, nil, err
	}
	config.User = fmt.Sprintf("cf:%s/0", appModel.Guid)

	sshCodeLines, err := a.CLI.CliCommandWithoutTerminalOutput("ssh-code")
	if err != nil {
		return nil, nil, err
	}
	config.Code = sshCodeLines[0]

	forwardedPort := firstForwardedServicePort
	for _, svcType := range serviceTypes(svcs) {
		for i, svc := range svcs[svcType] {
			if address := forward(svc.Credentials, forwardedPort); address != "" {
				config.Forwards = append(config.Forwards, service.Forward{
					Name: fmt.Sprintf("%s[%d]", svcType, i),
					From: fmt.Sprintf("localhost:%d", forwardedPort),
					To:   address,
				})
			} else {
				a.UI.Warn("unable to forward service index %d of type %s", i, svcType)
			}
			forwardedPort++
		}
	}
	return svcs, config, nil
}

func serviceTypes(s service.Services) (types []string) {
	for t := range s {
		types = append(types, t)
	}
	sort.Strings(types)
	return
}

func (a *App) sshEndpoint() (host, port string, err error) {
	target, err := a.CLI.ApiEndpoint()
	if err != nil {
		return "", "", err
	}
	url := fmt.Sprintf("%s/v2/info", target)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()

	var result struct {
		AppSSHEndpoint string `json:"app_ssh_endpoint"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", "", err
	}

	return net.SplitHostPort(result.AppSSHEndpoint)
}

func forward(creds map[string]string, toPort uint) (fromAddress string) {
	if creds == nil {
		return ""
	}

	forwardedPort := strconv.FormatUint(uint64(toPort), 10)
	override := map[string]string{}
	host, port := creds["hostname"], creds["port"]
	if host != "" || port != "" {
		override["port"] = forwardedPort
	}
	if host != "" {
		override["hostname"] = "localhost"
	}

	if creds["uri"] != "" {
		u, err := url.Parse(creds["uri"])
		if err != nil || u.Host == "" {
			return ""
		}
		host, port = ensureHostPort(host, port, u.Host)
		u.Host = "localhost:" + forwardedPort
		override["uri"] = u.String()
	}

	if creds["jdbcUrl"] != "" {
		u, err := url.Parse(strings.TrimPrefix(creds["jdbcUrl"], "jdbc:"))
		if err != nil || u.Host == "" {
			return ""
		}
		host, port = ensureHostPort(host, port, u.Host)
		u.Host = "localhost:" + forwardedPort
		override["jdbcUrl"] = "jdbc:" + u.String()
	}

	if host == "" || port == "" {
		return ""
	}
	merge(override, creds)
	return host + ":" + port
}

func ensureHostPort(knownHost, knownPort, address string) (host, port string) {
	if h, p, err := net.SplitHostPort(address); err == nil {
		host, port = h, p
	} else {
		host = address
	}
	if knownHost != "" {
		host = knownHost
	}
	if knownPort != "" {
		port = knownPort
	}
	return
}

func merge(from, to map[string]string) {
	for k, v := range from {
		to[k] = v
	}
}
