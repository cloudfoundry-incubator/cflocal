package remote

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

func (a *App) Droplet(name string) (droplet io.ReadCloser, size int64, err error) {
	return a.get(name, "/droplet/download")
}

func (a *App) SetDroplet(name string, droplet io.Reader, size int64) error {
	fieldname := "droplet"
	filename := fmt.Sprintf("%s.droplet", name)

	// This is necessary because (similar to S3) CC does not accept chunked multipart MIME
	contentLength := emptyMultipartSize(fieldname, filename) + size

	readBody, writeBody := io.Pipe()
	defer readBody.Close()

	form := multipart.NewWriter(writeBody)
	errChan := make(chan error, 1)
	go func() {
		defer writeBody.Close()

		dropletPart, err := form.CreateFormFile(fieldname, filename)
		if err != nil {
			errChan <- err
			return
		}
		if _, err := io.CopyN(dropletPart, droplet, size); err != nil {
			errChan <- err
			return
		}
		errChan <- form.Close()
	}()

	if err := a.putJob(name, "/droplet/upload", readBody, form.FormDataContentType(), contentLength); err != nil {
		<-errChan
		return err
	}

	return <-errChan
}

func emptyMultipartSize(fieldname, filename string) int64 {
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	form.CreateFormFile(fieldname, filename)
	form.Close()
	return int64(body.Len())
}

func (a *App) putJob(name, appEndpoint string, body io.Reader, contentType string, contentLength int64) error {
	response, err := a.doAppRequest(name, "PUT", appEndpoint, body, contentType, contentLength, http.StatusCreated)
	if err != nil {
		return err
	}
	return a.waitForJob(response.Body)
}

func (a *App) waitForJob(body io.ReadCloser) error {
	for {
		var job struct {
			Entity struct {
				GUID   string `json:"guid"`
				Status string `json:"status"`
			} `json:"entity"`
		}
		if err := decodeJob(body, &job); err != nil {
			return err
		}

		switch job.Entity.Status {
		case "queued", "running":
			endpoint := fmt.Sprintf("/v2/jobs/%s", job.Entity.GUID)
			response, err := a.doRequest("GET", endpoint, nil, "", 0, http.StatusOK)
			if err != nil {
				return err
			}
			body = response.Body
		case "finished":
			return nil
		default:
			return errors.New("job failed")
		}
	}
}

func decodeJob(body io.ReadCloser, job interface{}) error {
	defer body.Close()
	return json.NewDecoder(body).Decode(job)
}
