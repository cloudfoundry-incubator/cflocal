package remote

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	"github.com/sclevine/cflocal/service"
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

func (a *App) Droplet(name string) (droplet io.ReadCloser, size int64, err error) {
	return a.get(name, "/droplet/download")
}

func (a *App) SetDroplet(name string, droplet io.Reader) error {
	readBody, writeBody := io.Pipe()
	defer readBody.Close()

	form := multipart.NewWriter(writeBody)
	errChan := make(chan error, 1)
	go func() {
		defer writeBody.Close()

		dropletPart, err := form.CreateFormFile("droplet", name+".droplet")
		if err != nil {
			errChan <- err
			return
		}
		if _, err := io.Copy(dropletPart, droplet); err != nil {
			errChan <- err
			return
		}
		if err := form.Close(); err != nil {
			errChan <- err
			return
		}
	}()

	if err := a.put(name, "/droplet/upload", form.FormDataContentType(), readBody); err != nil {
		return err
	}

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
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
	return a.put(name, "", "application/json", body)
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

func (a *App) get(name, endpoint string) (io.ReadCloser, int64, error) {
	response, err := a.doRequest(name, "GET", endpoint, "", nil, http.StatusOK)
	if err != nil {
		return nil, 0, err
	}
	return response.Body, response.ContentLength, nil
}

func (a *App) put(name, endpoint, contentType string, body io.Reader) error {
	response, err := a.doRequest(name, "PUT", endpoint, contentType, body, http.StatusCreated)
	if err != nil {
		return err
	}
	response.Body.Close()
	return nil
}

func (a *App) doRequest(name, method, endpoint, contentType string, body io.Reader, desiredStatus int) (*http.Response, error) {
	guid, err := a.getGUID(name)
	if err != nil {
		return nil, err
	}
	target, err := a.CLI.ApiEndpoint()
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v2/apps/%s", target, path.Join(guid, endpoint))
	request, err := http.NewRequest(method, url, body)
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

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != desiredStatus {
		response.Body.Close()
		return nil, fmt.Errorf("unexpected '%s' from: %s %s", response.Status, method, url)
	}

	return response, nil
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
