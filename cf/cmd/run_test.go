package cmd_test

import (
	"io/ioutil"
	"time"

	"github.com/buildpack/forge"
	"github.com/buildpack/forge/app"
	"github.com/buildpack/forge/engine"
	"github.com/fatih/color"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "code.cloudfoundry.org/cflocal/cf/cmd"
	"code.cloudfoundry.org/cflocal/cf/cmd/mocks"
	sharedmocks "code.cloudfoundry.org/cflocal/mocks"
)

var _ = Describe("Run", func() {
	var (
		mockCtrl      *gomock.Controller
		mockUI        *sharedmocks.MockUI
		mockRunner    *mocks.MockRunner
		mockForwarder *mocks.MockForwarder
		mockRemoteApp *mocks.MockRemoteApp
		mockImage     *mocks.MockImage
		mockFS        *mocks.MockFS
		mockHelp      *mocks.MockHelp
		mockConfig    *mocks.MockConfig
		cmd           *Run
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = sharedmocks.NewMockUI()
		mockRunner = mocks.NewMockRunner(mockCtrl)
		mockForwarder = mocks.NewMockForwarder(mockCtrl)
		mockRemoteApp = mocks.NewMockRemoteApp(mockCtrl)
		mockImage = mocks.NewMockImage(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockHelp = mocks.NewMockHelp(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)
		cmd = &Run{
			UI:        mockUI,
			Runner:    mockRunner,
			Forwarder: mockForwarder,
			RemoteApp: mockRemoteApp,
			Image:     mockImage,
			FS:        mockFS,
			Help:      mockHelp,
			Config:    mockConfig,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Match", func() {
		It("should return true when the first argument is run", func() {
			Expect(cmd.Match([]string{"run"})).To(BeTrue())
			Expect(cmd.Match([]string{"not-run"})).To(BeFalse())
			Expect(cmd.Match([]string{})).To(BeFalse())
			Expect(cmd.Match(nil)).To(BeFalse())
		})
	})

	Describe("#Run", func() {
		It("should run a droplet", func() {
			mockUI.Progress = make(chan engine.Progress, 2)
			progress := make(chan engine.Progress, 2)
			progress <- mockProgress{Value: "some-progress-forward"}
			progress <- mockProgress{Value: "some-progress-run"}
			close(progress)
			droplet := sharedmocks.NewMockBuffer("some-droplet")
			services := forge.Services{"some": {{Name: "services"}}}
			forwardedServices := forge.Services{"some": {{Name: "forwarded-services"}}}
			restart := make(<-chan time.Time)
			watchDone := make(chan struct{})
			health := make(chan string, 3)
			forwardDone, forwardDoneCalls := sharedmocks.NewMockFunc()

			forwardConfig := &forge.ForwardDetails{
				Host: "some-ssh-host",
			}
			localYML := &app.YAML{
				Applications: []*forge.AppConfig{
					{Name: "some-other-app"},
					{
						Name:     "some-app",
						Env:      map[string]string{"a": "b"},
						Services: forge.Services{"some": {{Name: "overwritten-services"}}},
					},
				},
			}
			mockFS.EXPECT().Abs("some-dir").Return("some-abs-dir", nil)
			mockConfig.EXPECT().Load().Return(localYML, nil)
			mockFS.EXPECT().ReadFile("./some-app.droplet").Return(droplet, int64(100), nil)
			mockRemoteApp.EXPECT().Services("some-service-app").Return(services, nil)
			mockRemoteApp.EXPECT().Forward("some-forward-app", services).Return(forwardedServices, forwardConfig, nil)

			gomock.InOrder(
				mockFS.EXPECT().Watch("some-abs-dir", time.Second).Return(restart, watchDone, nil),
				mockImage.EXPECT().Pull(NetworkStack).Return(progress),
				mockForwarder.EXPECT().Forward(gomock.Any()).Return(health, forwardDone, "some-container-id", nil).Do(
					func(config *forge.ForwardConfig) {
						Expect(config.AppName).To(Equal("some-app"))
						Expect(config.Stack).To(Equal(NetworkStack))
						Expect(config.Color("some-text")).To(Equal(color.GreenString("some-text")))
						Expect(config.Details).To(Equal(forwardConfig))
						Expect(config.HostIP).To(Equal("0.0.0.0"))
						Expect(config.HostPort).To(Equal("3000"))
						Eventually(config.Wait).Should(Receive())
					},
				),
				mockImage.EXPECT().Pull(RunStack).Return(progress),
				mockRunner.EXPECT().Run(gomock.Any()).Return(int64(0), nil).Do(
					func(config *forge.RunConfig) {
						Expect(ioutil.ReadAll(config.Droplet)).To(Equal([]byte("some-droplet")))
						Expect(config.Stack).To(Equal(RunStack))
						Expect(config.AppDir).To(Equal("some-abs-dir"))
						Expect(config.OutputDir).To(Equal("/home/vcap"))
						Expect(config.WorkingDir).To(Equal("/home/vcap/app"))
						Expect(config.Restart).To(Equal(restart))
						Expect(config.Color("some-text")).To(Equal(color.GreenString("some-text")))
						Expect(config.AppConfig).To(Equal(&forge.AppConfig{
							Name:     "some-app",
							Env:      map[string]string{"a": "b"},
							Services: forwardedServices,
						}))
						Expect(config.NetworkConfig).To(Equal(&forge.NetworkConfig{
							ContainerID:   "some-container-id",
							ContainerPort: "8080",
							HostIP:        "0.0.0.0",
							HostPort:      "3000",
						}))
					},
				),
			)
			health <- "starting"
			health <- "starting"
			health <- "healthy"
			Expect(cmd.Run([]string{
				"run", "some-app",
				"-i", "0.0.0.0",
				"-p", "3000",
				"-d", "some-dir", "-w",
				"-s", "some-service-app",
				"-f", "some-forward-app",
			})).To(Succeed())
			Expect(forwardDoneCalls()).To(Equal(1))
			Expect(droplet.Result()).To(BeEmpty())
			Expect(watchDone).To(BeClosed())
			Expect(mockUI.Out).To(gbytes.Say("Running some-app on port 3000..."))
			Expect(mockUI.Progress).To(Receive(Equal(mockProgress{Value: "some-progress-forward"})))
			Expect(mockUI.Progress).To(Receive(Equal(mockProgress{Value: "some-progress-run"})))
		})

		// TODO: test app dir when app dir is unspecified (currently tested by integration)
		// TODO: test without watching
		// TODO: test -w without -d
		// TODO: test free port picker when port is unspecified (currently tested by integration)
		// TODO: test different combinations of -s and -f
		// TODO: test without -f
		// TODO: test timeout
		// TODO: test wait interval + done
		// TODO: test with -t and with both -t and -w
	})
})
