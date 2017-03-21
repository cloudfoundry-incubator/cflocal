package testutil

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/cli/plugin/models"

	"github.com/sclevine/cflocal/mocks"
)

type Request struct {
	Method        string
	Path          string
	Authenticated bool
	ContentType   string
	ContentLength int64
	Body          string
}

type Server struct {
	cli *mocks.MockCliConnection
}

func Serve(cli *mocks.MockCliConnection) *Server {
	return &Server{cli}
}

func (s *Server) Handle(auth bool, status int, response string) (*Request, Calls) {
	request := &Request{}
	var accessToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*request = Request{
			Method:        r.Method,
			Path:          r.URL.Path,
			Authenticated: auth && r.Header.Get("Authorization") == accessToken,
		}
		if r.Method == "PUT" || r.Method == "POST" {
			defer r.Body.Close()
			request.ContentType = r.Header.Get("Content-Type")
			request.ContentLength = r.ContentLength
			if body, err := ioutil.ReadAll(r.Body); err == nil {
				request.Body = string(body)
			}
		}
		w.WriteHeader(status)
		w.Write([]byte(response))
	}))
	calls := Calls{s.cli.EXPECT().ApiEndpoint().Return(server.URL, nil)}
	if auth {
		accessToken = "token for: " + server.URL
		calls = append(calls, s.cli.EXPECT().AccessToken().Return(accessToken, nil))
	}
	return request, calls
}

func (s *Server) HandleApp(name string, status int, response string) (*Request, Calls) {
	loginCall := s.cli.EXPECT().IsLoggedIn().Return(true, nil)
	getAppCall := s.cli.EXPECT().GetApp(name).Return(plugin_models.GetAppModel{Guid: "some-app-guid"}, nil).After(loginCall)

	request, calls := s.Handle(true, status, response)
	calls.AfterCall(loginCall)

	return request, append(calls, loginCall, getAppCall)
}
