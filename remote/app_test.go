package remote_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/sclevine/cflocal/mocks"
	. "github.com/sclevine/cflocal/remote"

	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("App", func() {
	var (
		mockCtrl *gomock.Controller
		mockCLI  *mocks.MockCliConnection
		app      *App
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCLI = mocks.NewMockCliConnection(mockCtrl)
		app = &App{CLI: mockCLI}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	type request struct {
		authToken string
		path      string
	}
	handleAppEndpoint := func(name, response string) *request {
		req := &request{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			req.authToken = r.Header.Get("Authorization")
			req.path = r.URL.Path
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

	Describe("#Droplet", func() {
		It("should return the app's droplet", func() {
			req := handleAppEndpoint("some-name", "some-droplet")

			droplet, size, err := app.Droplet("some-name")
			Expect(err).NotTo(HaveOccurred())
			defer droplet.Close()

			Expect(size).To(Equal(int64(12)))
			Expect(ioutil.ReadAll(droplet)).To(Equal([]byte("some-droplet")))
			Expect(req.authToken).To(Equal("some-access-token"))
			Expect(req.path).To(Equal("/v2/apps/some-guid/droplet/download"))
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
			Expect(req.authToken).To(Equal("some-access-token"))
			Expect(req.path).To(Equal("/v2/apps/some-guid/env"))
		})
	})
})
