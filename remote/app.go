package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"

	cfplugin "code.cloudfoundry.org/cli/plugin"
)

type App struct {
	CLI cfplugin.CliConnection
}

func (a *App) Droplet(name string) (droplet io.ReadCloser, size int64, err error) {
	return a.get(name, "/droplet/download")
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

type AppEnv struct {
	Staging map[string]string `json:"staging_env_json"`
	Running map[string]string `json:"running_env_json"`
	App     map[string]string `json:"environment_json"`
}

func (a *App) get(name, endpoint string) (io.ReadCloser, int64, error) {
	loggedIn, err := a.CLI.IsLoggedIn()
	if err != nil {
		return nil, 0, err
	}
	if !loggedIn {
		return nil, 0, errors.New("must be authenticated with API")
	}
	model, err := a.CLI.GetApp(name)
	if err != nil {
		return nil, 0, err
	}
	target, err := a.CLI.ApiEndpoint()
	if err != nil {
		return nil, 0, err
	}
	url := fmt.Sprintf("%s/v2/apps/%s", target, path.Join(model.Guid, endpoint))
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
