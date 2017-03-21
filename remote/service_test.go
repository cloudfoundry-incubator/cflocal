package remote_test

import (
	"net/http"

	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/sclevine/cflocal/mocks"
	. "github.com/sclevine/cflocal/remote"
	"github.com/sclevine/cflocal/service"
	"github.com/sclevine/cflocal/testutil"
)

var _ = Describe("App - Service", func() {
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
		app = &App{CLI: mockCLI, UI: mockUI}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Services", func() {
		It("should return the app's services", func() {
			req, _ := server.HandleApp("some-name", http.StatusOK, `{
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
			Expect(app.Services("some-name")).To(Equal(service.Services{
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
			Expect(req.Method).To(Equal("GET"))
			Expect(req.Path).To(Equal("/v2/apps/some-app-guid/env"))
			Expect(req.Authenticated).To(BeTrue())
		})
	})

	Describe("#Forward", func() {
		It("should translate the provided services to forwarded services", func() {
			req, _ := server.Handle(false, http.StatusOK, `{"app_ssh_endpoint": "some-ssh-host:1000"}`)
			testutil.Calls{
				mockCLI.EXPECT().GetApp("some-name").Return(plugin_models.GetAppModel{Guid: "some-guid"}, nil),
				mockCLI.EXPECT().CliCommandWithoutTerminalOutput("ssh-code").Return([]string{"some-code", "something-else"}, nil),
			}.AfterCall(mockCLI.EXPECT().IsLoggedIn().Return(true, nil))

			services, config, err := app.Forward("some-name", service.Services{
				"common": {
					{
						Name:  "some-name-0",
						Label: "some-label",
						Tags:  []string{"some", "tags"},
						Plan:  "some-plan",
						Credentials: map[string]string{
							"hostname": "some-host",
							"port":     "3306",
							"uri":      "mysql://some-user:some-password@some-host:3306/some-db?reconnect=true",
							"jdbcUrl":  "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
							"some-key": "some-value",
						},
						SyslogDrainURL: strPtr("some-url"),
						Provider:       strPtr("some-provider"),
						VolumeMounts:   []string{"some", "mounts"},
					},
				},
				"full-url": {
					{
						Name: "some-name-1",
						Credentials: map[string]string{
							"hostname": "some-host",
							"port":     "3306",
							"uri":      "mysql://some-user:some-password@some-host:3306/some-db?reconnect=true",
							"jdbcUrl":  "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-2",
						Credentials: map[string]string{
							"hostname": "some-host",
							"uri":      "mysql://some-user:some-password@some-host:3306/some-db?reconnect=true",
						},
					},
					{
						Name: "some-name-3",
						Credentials: map[string]string{
							"port":    "3306",
							"jdbcUrl": "jdbc:mysql://some-host:3306/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-4",
						Credentials: map[string]string{
							"uri":     "mysql://some-user:some-password@some-host:3306/some-db?reconnect=true",
							"jdbcUrl": "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
				},
				"host-url": {
					{
						Name: "some-name-5",
						Credentials: map[string]string{
							"hostname": "some-host",
							"port":     "3306",
							"uri":      "mysql://some-user:some-password@some-host/some-db?reconnect=true",
							"jdbcUrl":  "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-6",
						Credentials: map[string]string{
							"hostname": "some-host",
							"uri":      "mysql://some-user:some-password@some-host/some-db?reconnect=true",
						},
					},
					{
						Name: "some-name-7",
						Credentials: map[string]string{
							"port":    "3306",
							"jdbcUrl": "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-8",
						Credentials: map[string]string{
							"uri":     "mysql://some-user:some-password@some-host/some-db?reconnect=true",
							"jdbcUrl": "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
				},
				"no-url": {
					{
						Name: "some-name-9",
						Credentials: map[string]string{
							"hostname": "some-host",
							"port":     "3306",
						},
					},
					{
						Name: "some-name-10",
						Credentials: map[string]string{
							"hostname": "some-host",
						},
					},
					{
						Name: "some-name-11",
						Credentials: map[string]string{
							"port": "3306",
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(Equal(service.Services{
				"common": {
					{
						Name:  "some-name-0",
						Label: "some-label",
						Tags:  []string{"some", "tags"},
						Plan:  "some-plan",
						Credentials: map[string]string{
							"hostname": "localhost",
							"port":     "40000",
							"uri":      "mysql://some-user:some-password@localhost:40000/some-db?reconnect=true",
							"jdbcUrl":  "jdbc:mysql://localhost:40000/some-db?user=some-user\u0026password=some-password",
							"some-key": "some-value",
						},
						SyslogDrainURL: strPtr("some-url"),
						Provider:       strPtr("some-provider"),
						VolumeMounts:   []string{"some", "mounts"},
					},
				},
				"full-url": {
					{
						Name: "some-name-1",
						Credentials: map[string]string{
							"hostname": "localhost",
							"port":     "40001",
							"uri":      "mysql://some-user:some-password@localhost:40001/some-db?reconnect=true",
							"jdbcUrl":  "jdbc:mysql://localhost:40001/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-2",
						Credentials: map[string]string{
							"hostname": "localhost",
							"port":     "40002",
							"uri":      "mysql://some-user:some-password@localhost:40002/some-db?reconnect=true",
						},
					},
					{
						Name: "some-name-3",
						Credentials: map[string]string{
							"port":    "40003",
							"jdbcUrl": "jdbc:mysql://localhost:40003/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-4",
						Credentials: map[string]string{
							"uri":     "mysql://some-user:some-password@localhost:40004/some-db?reconnect=true",
							"jdbcUrl": "jdbc:mysql://localhost:40004/some-db?user=some-user\u0026password=some-password",
						},
					},
				},
				"host-url": {
					{
						Name: "some-name-5",
						Credentials: map[string]string{
							"hostname": "localhost",
							"port":     "40005",
							"uri":      "mysql://some-user:some-password@localhost:40005/some-db?reconnect=true",
							"jdbcUrl":  "jdbc:mysql://localhost:40005/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-6",
						Credentials: map[string]string{
							"hostname": "some-host",
							"uri":      "mysql://some-user:some-password@some-host/some-db?reconnect=true",
						},
					},
					{
						Name: "some-name-7",
						Credentials: map[string]string{
							"port":    "40007",
							"jdbcUrl": "jdbc:mysql://localhost:40007/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-8",
						Credentials: map[string]string{
							"uri":     "mysql://some-user:some-password@some-host/some-db?reconnect=true",
							"jdbcUrl": "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
				},
				"no-url": {
					{
						Name: "some-name-9",
						Credentials: map[string]string{
							"hostname": "localhost",
							"port":     "40009",
						},
					},
					{
						Name: "some-name-10",
						Credentials: map[string]string{
							"hostname": "some-host",
						},
					},
					{
						Name: "some-name-11",
						Credentials: map[string]string{
							"port": "3306",
						},
					},
				},
			}))
			Expect(config).To(Equal(&service.ForwardConfig{
				Host: "some-ssh-host",
				Port: "1000",
				User: "cf:some-guid/0",
				Code: "some-code",
				Forwards: []service.Forward{
					{
						Name: "common[0]",
						From: "localhost:40000",
						To:   "some-host:3306",
					},
					{
						Name: "full-url[0]",
						From: "localhost:40001",
						To:   "some-host:3306",
					},
					{
						Name: "full-url[1]",
						From: "localhost:40002",
						To:   "some-host:3306",
					},
					{
						Name: "full-url[2]",
						From: "localhost:40003",
						To:   "some-host:3306",
					},
					{
						Name: "full-url[3]",
						From: "localhost:40004",
						To:   "some-host:3306",
					},
					{
						Name: "host-url[0]",
						From: "localhost:40005",
						To:   "some-host:3306",
					},
					{
						Name: "host-url[2]",
						From: "localhost:40007",
						To:   "some-host:3306",
					},
					{
						Name: "no-url[0]",
						From: "localhost:40009",
						To:   "some-host:3306",
					},
				},
			}))

			Expect(mockUI.Out).To(gbytes.Say("Warning: unable to forward service index 1 of type host-url"))
			Expect(mockUI.Out).To(gbytes.Say("Warning: unable to forward service index 3 of type host-url"))
			Expect(mockUI.Out).To(gbytes.Say("Warning: unable to forward service index 1 of type no-url"))

			Expect(req.Method).To(Equal("GET"))
			Expect(req.Path).To(Equal("/v2/info"))
			Expect(req.Authenticated).To(BeFalse())
		})
	})
})

func strPtr(s string) *string {
	return &s
}
