package remote_test

import (
	"net/http"

	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"code.cloudfoundry.org/cflocal/mocks"
	. "code.cloudfoundry.org/cflocal/remote"
	"code.cloudfoundry.org/cflocal/testutil"
	"github.com/sclevine/forge"
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
		app = &App{CLI: mockCLI, UI: mockUI, HTTP: &http.Client{}}
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
			Expect(app.Services("some-name")).To(Equal(forge.Services{
				"some-type": {
					{
						Name:           "some-name",
						Label:          "some-label",
						Tags:           []string{"some", "tags"},
						Plan:           "some-plan",
						Credentials:    map[string]interface{}{"some": "credentials"},
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
			gomock.InOrder(
				mockCLI.EXPECT().IsLoggedIn().Return(true, nil),
				mockCLI.EXPECT().GetApp("some-name").Return(plugin_models.GetAppModel{Guid: "some-guid"}, nil),
			)

			services, config, err := app.Forward("some-name", forge.Services{
				"common": {
					{
						Name:  "some-name-0",
						Label: "some-label",
						Tags:  []string{"some", "tags"},
						Plan:  "some-plan",
						Credentials: map[string]interface{}{
							"hostname": "some-host",
							"port":     float64(3306),
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
						Credentials: map[string]interface{}{
							"hostname": "some-host",
							"port":     float64(3306),
							"uri":      "mysql://some-user:some-password@some-host:3306/some-db?reconnect=true",
							"jdbcUrl":  "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-2",
						Credentials: map[string]interface{}{
							"hostname": "some-host",
							"uri":      "mysql://some-user:some-password@some-host:3306/some-db?reconnect=true",
						},
					},
					{
						Name: "some-name-3",
						Credentials: map[string]interface{}{
							"port":    float64(3306),
							"jdbcUrl": "jdbc:mysql://some-host:3306/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-4",
						Credentials: map[string]interface{}{
							"uri":     "mysql://some-user:some-password@some-host:3306/some-db?reconnect=true",
							"jdbcUrl": "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
				},
				"host-url": {
					{
						Name: "some-name-5",
						Credentials: map[string]interface{}{
							"hostname": "some-host",
							"port":     float64(3306),
							"uri":      "mysql://some-user:some-password@some-host/some-db?reconnect=true",
							"jdbcUrl":  "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-6",
						Credentials: map[string]interface{}{
							"hostname": "some-host",
							"uri":      "mysql://some-user:some-password@some-host/some-db?reconnect=true",
						},
					},
					{
						Name: "some-name-7",
						Credentials: map[string]interface{}{
							"port":    float64(3306),
							"jdbcUrl": "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-8",
						Credentials: map[string]interface{}{
							"uri":     "mysql://some-user:some-password@some-host/some-db?reconnect=true",
							"jdbcUrl": "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
				},
				"no-url": {
					{
						Name: "some-name-9",
						Credentials: map[string]interface{}{
							"hostname": "some-host",
							"port":     float64(3306),
						},
					},
					{
						Name: "some-name-10",
						Credentials: map[string]interface{}{
							"hostname": "some-host",
						},
					},
					{
						Name: "some-name-11",
						Credentials: map[string]interface{}{
							"port": float64(3306),
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(Equal(forge.Services{
				"common": {
					{
						Name:  "some-name-0",
						Label: "some-label",
						Tags:  []string{"some", "tags"},
						Plan:  "some-plan",
						Credentials: map[string]interface{}{
							"hostname": "localhost",
							"port":     float64(40000),
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
						Credentials: map[string]interface{}{
							"hostname": "localhost",
							"port":     float64(40001),
							"uri":      "mysql://some-user:some-password@localhost:40001/some-db?reconnect=true",
							"jdbcUrl":  "jdbc:mysql://localhost:40001/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-2",
						Credentials: map[string]interface{}{
							"hostname": "localhost",
							"port":     float64(40002),
							"uri":      "mysql://some-user:some-password@localhost:40002/some-db?reconnect=true",
						},
					},
					{
						Name: "some-name-3",
						Credentials: map[string]interface{}{
							"port":    float64(40003),
							"jdbcUrl": "jdbc:mysql://localhost:40003/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-4",
						Credentials: map[string]interface{}{
							"uri":     "mysql://some-user:some-password@localhost:40004/some-db?reconnect=true",
							"jdbcUrl": "jdbc:mysql://localhost:40004/some-db?user=some-user\u0026password=some-password",
						},
					},
				},
				"host-url": {
					{
						Name: "some-name-5",
						Credentials: map[string]interface{}{
							"hostname": "localhost",
							"port":     float64(40005),
							"uri":      "mysql://some-user:some-password@localhost:40005/some-db?reconnect=true",
							"jdbcUrl":  "jdbc:mysql://localhost:40005/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-6",
						Credentials: map[string]interface{}{
							"hostname": "some-host",
							"uri":      "mysql://some-user:some-password@some-host/some-db?reconnect=true",
						},
					},
					{
						Name: "some-name-7",
						Credentials: map[string]interface{}{
							"port":    float64(40007),
							"jdbcUrl": "jdbc:mysql://localhost:40007/some-db?user=some-user\u0026password=some-password",
						},
					},
					{
						Name: "some-name-8",
						Credentials: map[string]interface{}{
							"uri":     "mysql://some-user:some-password@some-host/some-db?reconnect=true",
							"jdbcUrl": "jdbc:mysql://some-host/some-db?user=some-user\u0026password=some-password",
						},
					},
				},
				"no-url": {
					{
						Name: "some-name-9",
						Credentials: map[string]interface{}{
							"hostname": "localhost",
							"port":     float64(40009),
						},
					},
					{
						Name: "some-name-10",
						Credentials: map[string]interface{}{
							"hostname": "some-host",
						},
					},
					{
						Name: "some-name-11",
						Credentials: map[string]interface{}{
							"port": float64(3306),
						},
					},
				},
			}))
			Expect(config.Host).To(Equal("some-ssh-host"))
			Expect(config.Port).To(Equal("1000"))
			Expect(config.User).To(Equal("cf:some-guid/0"))
			Expect(config.Forwards).To(Equal([]forge.Forward{
				{
					Name: "some-name-0:common[0]",
					From: "40000",
					To:   "some-host:3306",
				},
				{
					Name: "some-name-1:full-url[0]",
					From: "40001",
					To:   "some-host:3306",
				},
				{
					Name: "some-name-2:full-url[1]",
					From: "40002",
					To:   "some-host:3306",
				},
				{
					Name: "some-name-3:full-url[2]",
					From: "40003",
					To:   "some-host:3306",
				},
				{
					Name: "some-name-4:full-url[3]",
					From: "40004",
					To:   "some-host:3306",
				},
				{
					Name: "some-name-5:host-url[0]",
					From: "40005",
					To:   "some-host:3306",
				},
				{
					Name: "some-name-7:host-url[2]",
					From: "40007",
					To:   "some-host:3306",
				},
				{
					Name: "some-name-9:no-url[0]",
					From: "40009",
					To:   "some-host:3306",
				},
			}))

			Expect(mockUI.Out).To(gbytes.Say(`Warning: unable to forward service: some-name-6:host-url\[1\]`))
			Expect(mockUI.Out).To(gbytes.Say(`Warning: unable to forward service: some-name-8:host-url\[3\]`))
			Expect(mockUI.Out).To(gbytes.Say(`Warning: unable to forward service: some-name-10:no-url\[1\]`))

			Expect(req.Method).To(Equal("GET"))
			Expect(req.Path).To(Equal("/v2/info"))
			Expect(req.Authenticated).To(BeFalse())

			mockCLI.EXPECT().CliCommandWithoutTerminalOutput("ssh-code").Return([]string{"some-code-1", "something-else"}, nil)
			mockCLI.EXPECT().CliCommandWithoutTerminalOutput("ssh-code").Return([]string{"some-code-2", "something-else"}, nil)
			Expect(config.Code()).To(Equal("some-code-1"))
			Expect(config.Code()).To(Equal("some-code-2"))
		})

		// TODO: test no valid forwards
	})
})

func strPtr(s string) *string {
	return &s
}
