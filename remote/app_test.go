package remote_test

import (
	"net/http"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sclevine/cflocal/mocks"
	. "github.com/sclevine/cflocal/remote"
	"github.com/sclevine/cflocal/testutil"
)

var _ = Describe("App", func() {
	var (
		mockCtrl *gomock.Controller
		mockCLI  *mocks.MockCliConnection
		mockUI   *mocks.MockUI
		server   *testutil.Server
		app      *App
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCLI = mocks.NewMockCliConnection(mockCtrl)
		mockUI = mocks.NewMockUI()
		server = testutil.Serve(mockCLI)
		app = &App{CLI: mockCLI, UI: mockUI, HTTP: &http.Client{}}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Command", func() {
		It("should return the app's environment variables", func() {
			req, _ := server.HandleApp("some-name", http.StatusOK, `{
				"entity": {"command": "some-command"}
			}`)

			Expect(app.Command("some-name")).To(Equal("some-command"))

			Expect(req.Method).To(Equal("GET"))
			Expect(req.Path).To(Equal("/v2/apps/some-app-guid"))
			Expect(req.Authenticated).To(BeTrue())
		})
	})

	Describe("#Env", func() {
		It("should return the app's environment variables", func() {
			req, _ := server.HandleApp("some-name", http.StatusOK, `{
				"staging_env_json": {"a": "b", "c": "d"},
				"running_env_json": {"e": "f", "g": "h"},
				"environment_json": {"i": "j", "k": "l"}
			}`)

			Expect(app.Env("some-name")).To(Equal(&AppEnv{
				Staging: map[string]string{"a": "b", "c": "d"},
				Running: map[string]string{"e": "f", "g": "h"},
				App:     map[string]string{"i": "j", "k": "l"},
			}))

			Expect(req.Method).To(Equal("GET"))
			Expect(req.Path).To(Equal("/v2/apps/some-app-guid/env"))
			Expect(req.Authenticated).To(BeTrue())

		})
	})

	Describe("#SetEnv", func() {
		It("should set the app's environment", func() {
			req, _ := server.HandleApp("some-name", http.StatusCreated, "{}")

			Expect(app.SetEnv("some-name", map[string]string{"some-key": "some-value"})).To(Succeed())

			Expect(req.Method).To(Equal("PUT"))
			Expect(req.Path).To(Equal("/v2/apps/some-app-guid"))
			Expect(req.Authenticated).To(BeTrue())
			Expect(req.ContentType).To(Equal("application/x-www-form-urlencoded"))
			Expect(req.ContentLength).To(Equal(int64(47)))
			Expect(req.Body).To(MatchJSON(`{
				"environment_json": {
					"some-key": "some-value"
				}
			}`))
		})
	})

	Describe("#Restart", func() {
		It("should restart the app", func() {
			mockCLI.EXPECT().CliCommand("restart", "some-name").Return(nil, nil)
			Expect(app.Restart("some-name")).To(Succeed())
		})
	})
})
