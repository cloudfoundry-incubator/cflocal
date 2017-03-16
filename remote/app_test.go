package remote_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/sclevine/cflocal/mocks"
	. "github.com/sclevine/cflocal/remote"

	"bytes"

	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/cflocal/service"
)

var _ = Describe("App", func() {
	var (
		mockCtrl *gomock.Controller
		mockCLI  *mocks.MockCliConnection
		mockUI   *mocks.MockUI
		app      *App
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCLI = mocks.NewMockCliConnection(mockCtrl)
		mockUI = mocks.NewMockUI()
		app = &App{CLI: mockCLI, UI: mockUI}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	type request struct {
		method      string
		path        string
		contentType string
		authToken   string
		body        string
	}
	handleAppEndpoint := func(name, response string) *request {
		req := &request{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			req.method = r.Method
			req.path = r.URL.Path
			req.contentType = r.Header.Get("Content-Type")
			req.authToken = r.Header.Get("Authorization")
			if body, err := ioutil.ReadAll(r.Body); err == nil && len(body) > 0 {
				req.body = string(body)
				w.WriteHeader(http.StatusCreated)
			}
			w.Write([]byte(response))
		}))
		gomock.InOrder(
			mockCLI.EXPECT().IsLoggedIn().Return(true, nil),
			mockCLI.EXPECT().GetApp(name).Return(plugin_models.GetAppModel{Guid: "some-guid"}, nil),
			mockCLI.EXPECT().ApiEndpoint().Return(server.URL, nil),
			mockCLI.EXPECT().AccessToken().Return("some-access-token", nil),
		)
		return req
	}

	Describe("#Command", func() {
		It("should return the app's environment variables", func() {
			req := handleAppEndpoint("some-name", `{
				"entity": {"command": "some-command"}
			}`)
			command, err := app.Command("some-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(command).To(Equal("some-command"))

			Expect(req.method).To(Equal("GET"))
			Expect(req.path).To(Equal("/v2/apps/some-guid"))
			Expect(req.authToken).To(Equal("some-access-token"))
		})
	})

	Describe("#Droplet", func() {
		It("should return the app's droplet", func() {
			req := handleAppEndpoint("some-name", "some-droplet")

			droplet, size, err := app.Droplet("some-name")
			Expect(err).NotTo(HaveOccurred())
			defer droplet.Close()

			Expect(size).To(Equal(int64(12)))
			Expect(ioutil.ReadAll(droplet)).To(Equal([]byte("some-droplet")))

			Expect(req.method).To(Equal("GET"))
			Expect(req.path).To(Equal("/v2/apps/some-guid/droplet/download"))
			Expect(req.authToken).To(Equal("some-access-token"))
		})
	})

	Describe("#SetDroplet", func() {
		It("should upload the app's droplet", func() {
			req := handleAppEndpoint("some-name", "")

			Expect(app.SetDroplet("some-name", bytes.NewBufferString("some-droplet"))).To(Succeed())

			Expect(req.method).To(Equal("PUT"))
			Expect(req.path).To(Equal("/v2/apps/some-guid/droplet/upload"))
			Expect(req.contentType).To(MatchRegexp("multipart/form-data; boundary=[A-Fa-f0-9]{60}"))
			Expect(req.authToken).To(Equal("some-access-token"))
			boundary := req.contentType[len(req.contentType)-60:]
			Expect(req.body).To(MatchRegexp(
				`--%s\s+Content-Disposition: form-data; name="droplet"; filename="some-name\.droplet"\s+`+
					`Content-Type: application/octet-stream\s+some-droplet\s+--%[1]s--`, boundary,
			))
		})
	})

	Describe("#Env", func() {
		It("should return the app's environment variables", func() {
			req := handleAppEndpoint("some-name", `{
				"staging_env_json": {"a": "b", "c": "d"},
				"running_env_json": {"e": "f", "g": "h"},
				"environment_json": {"i": "j", "k": "l"}
			}`)

			env, err := app.Env("some-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(env).To(Equal(&AppEnv{
				Staging: map[string]string{"a": "b", "c": "d"},
				Running: map[string]string{"e": "f", "g": "h"},
				App:     map[string]string{"i": "j", "k": "l"},
			}))

			Expect(req.method).To(Equal("GET"))
			Expect(req.path).To(Equal("/v2/apps/some-guid/env"))
			Expect(req.authToken).To(Equal("some-access-token"))
		})
	})

	Describe("#SetEnv", func() {
		It("should set the app's environment", func() {
			req := handleAppEndpoint("some-name", "")

			Expect(app.SetEnv("some-name", map[string]string{"some-key": "some-value"})).To(Succeed())

			Expect(req.method).To(Equal("PUT"))
			Expect(req.path).To(Equal("/v2/apps/some-guid"))
			Expect(req.contentType).To(Equal("application/json"))
			Expect(req.authToken).To(Equal("some-access-token"))
			Expect(req.body).To(MatchJSON(`{
				"environment_json": {
					"some-key": "some-value"
				}
			}`))
		})
	})

	Describe("#Services", func() {
		It("should return the app's services", func() {
			req := handleAppEndpoint("some-name", `{
				"system_env_json": {
					"VCAP_SERVICES": {
						"some-type": [{
							"name": "some-name",
							"label": "some-label",
							"tags": ["some", "tags"],
							"plan": "some-plan",
							"credentials": {"some": "credentials"},
							"syslog_drain_url": "some-url",
							"provider": "some-provider",
							"volume_mounts": ["some", "mounts"]
						}]
					}
				}
			}`)

			services, err := app.Services("some-name")
			Expect(err).NotTo(HaveOccurred())

			Expect(services).To(Equal(service.Services{
				"some-type": {
					{
						Name:           "some-name",
						Label:          "some-label",
						Tags:           []string{"some", "tags"},
						Plan:           "some-plan",
						Credentials:    map[string]string{"some": "credentials"},
						SyslogDrainURL: strPtr("some-url"),
						Provider:       strPtr("some-provider"),
						VolumeMounts:   []string{"some", "mounts"},
					},
				},
			}))
			Expect(req.method).To(Equal("GET"))
			Expect(req.path).To(Equal("/v2/apps/some-guid/env"))
			Expect(req.authToken).To(Equal("some-access-token"))
		})
	})
})

func strPtr(s string) *string {
	return &s
}
