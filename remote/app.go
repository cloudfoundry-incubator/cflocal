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
	"github.com/sclevine/cflocal/service"
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
	appGUID, err := a.getGUID(name)
	if err != nil {
		return nil, nil, err
	}
	config.User = fmt.Sprintf("cf:%s/0", appGUID)

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
				a.UI.Warn("unable to forward service of type: %s", svcType)
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
