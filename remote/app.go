package remote

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	cfplugin "code.cloudfoundry.org/cli/plugin"
)

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

func (a *App) SetEnv(name string, env map[string]string) error {
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(struct {
		Env map[string]string `json:"environment_json"`
	}{env}); err != nil {
		return err
	}
	return a.put(name, "", body, "application/x-www-form-urlencoded", int64(body.Len()))
}

func (a *App) Restart(name string) error {
	_, err := a.CLI.CliCommand("restart", name)
	return err
}

func (a *App) get(name, appEndpoint string) (body io.ReadCloser, size int64, err error) {
	response, err := a.doAppRequest(name, "GET", appEndpoint, nil, "", 0, http.StatusOK)
	if err != nil {
		return nil, 0, err
	}
	return response.Body, response.ContentLength, nil
}

func (a *App) put(name, appEndpoint string, body io.Reader, contentType string, contentLength int64) error {
	response, err := a.doAppRequest(name, "PUT", appEndpoint, body, contentType, contentLength, http.StatusCreated)
	if err != nil {
		return err
	}
	return response.Body.Close()
}

func (a *App) doAppRequest(name, method, appEndpoint string, body io.Reader, contentType string, contentLength int64, desiredStatus int) (*http.Response, error) {
	if err := a.checkAuth(); err != nil {
		return nil, err
	}
	appModel, err := a.CLI.GetApp(name)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("/v2/apps/%s", path.Join(appModel.Guid, appEndpoint))
	response, err := a.doRequest(method, endpoint, body, contentType, contentLength, desiredStatus)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (a *App) doRequest(method, endpoint string, body io.Reader, contentType string, contentLength int64, desiredStatus int) (*http.Response, error) {
	target, err := a.CLI.ApiEndpoint()
	if err != nil {
		return nil, err
	}
	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	targetURL.Path = path.Join(targetURL.Path, endpoint)
	request, err := http.NewRequest(method, targetURL.String(), body)
	if err != nil {
		return nil, err
	}
	token, err := a.CLI.AccessToken()
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", token)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if contentLength > 0 {
		request.ContentLength = contentLength
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != desiredStatus {
		response.Body.Close()
		return nil, fmt.Errorf("unexpected '%s' from: %s %s", response.Status, method, targetURL.String())
	}
	return response, nil
}

func (a *App) checkAuth() error {
	loggedIn, err := a.CLI.IsLoggedIn()
	if err != nil {
		return err
	}
	if !loggedIn {
		return errors.New("must be authenticated with Cloud Foundry API")
	}
	return nil
}
