package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path"
	"sort"

	cfplugin "code.cloudfoundry.org/cli/plugin"
)

const firstForwardedServicePort uint = 40000

type App struct {
	CLI cfplugin.CliConnection
	UI  UI
}

type UI interface {
	Warn(format string, a ...interface{})
}

type AppEnv struct {
	Staging map[string]string `json:"staging_env_json"`
	Running map[string]string `json:"running_env_json"`
	App     map[string]string `json:"environment_json"`
}

type Service struct {
	Name           string            `json:"name" yaml:"name"`
	Label          string            `json:"label" yaml:"label"`
	Tags           []string          `json:"tags" yaml:"tags"`
	Plan           string            `json:"plan" yaml:"plan"`
	Credentials    map[string]string `json:"credentials" yaml:"credentials"`
	SyslogDrainURL *string           `json:"syslog_drain_url" yaml:"syslog_drain_url,omitempty"`
	Provider       *string           `json:"provider" yaml:"provider,omitempty"`
	VolumeMounts   []string          `json:"volume_mounts" yaml:"volume_mounts,omitempty"`
}

type Services map[string][]Service

func (a *App) Droplet(name string) (droplet io.ReadCloser, size int64, err error) {
	return a.get(name, "/droplet/download")
}

func (a *App) Command(name string) (string, error) {
	appJSON, _, err := a.get(name, "")
	if err != nil {
		return "", err
	}
	defer appJSON.Close()

	var app struct{ Entity struct{ Command string } }
	if err := json.NewDecoder(appJSON).Decode(&app); err != nil {
		return "", err
	}
	return app.Entity.Command, nil
}

func (a *App) Env(name string) (*AppEnv, error) {
	appEnvJSON, _, err := a.get(name, "/env")
	if err != nil {
		return nil, err
	}
	defer appEnvJSON.Close()
	var env AppEnv
	if err := json.NewDecoder(appEnvJSON).Decode(&env); err != nil {
		return nil, err
	}
	return &env, nil
}

func (a *App) Services(name string) (Services, error) {
	appEnvJSON, _, err := a.get(name, "/env")
	if err != nil {
		return nil, err
	}
	defer appEnvJSON.Close()
	var env struct {
		SystemEnvJSON struct {
			VCAPServices Services `json:"VCAP_SERVICES"`
		} `json:"system_env_json"`
	}
	if err := json.NewDecoder(appEnvJSON).Decode(&env); err != nil {
		return nil, err
	}
	return env.SystemEnvJSON.VCAPServices, nil
}

func (a *App) Forward(name string, services Services) (forwarded Services, command string, err error) {
	sshHost, sshPort, err := a.sshEndpoint()
	if err != nil {
		return nil, "", err
	}
	appGUID, err := a.getGUID(name)
	if err != nil {
		return nil, "", err
	}
	sshCodeLines, err := a.CLI.CliCommandWithoutTerminalOutput("ssh-code")
	if err != nil {
		return nil, "", err
	}

	needsForward := false
	forwardedPort := firstForwardedServicePort
	sshOptions := "-f -N -o StrictHostKeyChecking=no -o ExitOnForwardFailure=yes"
	command = fmt.Sprintf("sshpass -p %s %s -p %s cf:%s/0@%s", sshCodeLines[0], sshOptions, sshPort, appGUID, sshHost)
	for _, serviceType := range serviceTypes(services) {
		for _, service := range services[serviceType] {
			if address := forward(service.Credentials, forwardedPort); address != "" {
				command += fmt.Sprintf(" -L localhost:%d:%s", forwardedPort, address)
				needsForward = true
			} else {
				a.UI.Warn("unable to forward service of type: %s", serviceType)
			}
			forwardedPort++
		}
	}
	if !needsForward {
		command = ""
	}
	return services, command, nil
}

func serviceTypes(s Services) (types []string) {
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

	token, err := a.CLI.AccessToken()
	if err != nil {
		return "", "", err
	}
	request.Header.Add("Authorization", token)

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

func (a *App) getGUID(name string) (string, error) {
	loggedIn, err := a.CLI.IsLoggedIn()
	if err != nil {
		return "", err
	}
	if !loggedIn {
		return "", errors.New("must be authenticated with API")
	}
	model, err := a.CLI.GetApp(name)
	if err != nil {
		return "", err
	}
	return model.Guid, nil
}

func (a *App) get(name, endpoint string) (io.ReadCloser, int64, error) {
	guid, err := a.getGUID(name)
	if err != nil {
		return nil, 0, err
	}
	target, err := a.CLI.ApiEndpoint()
	if err != nil {
		return nil, 0, err
	}
	url := fmt.Sprintf("%s/v2/apps/%s", target, path.Join(guid, endpoint))
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}

	token, err := a.CLI.AccessToken()
	if err != nil {
		return nil, 0, err
	}
	request.Header.Add("Authorization", token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	return response.Body, response.ContentLength, nil
}
