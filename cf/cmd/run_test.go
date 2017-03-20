package cmd_test

import (
	"github.com/fatih/color"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/sclevine/cflocal/cf/cmd"
	"github.com/sclevine/cflocal/cf/cmd/mocks"
	"github.com/sclevine/cflocal/local"
	sharedmocks "github.com/sclevine/cflocal/mocks"
	"github.com/sclevine/cflocal/service"
)

var _ = Describe("Run", func() {
	var (
		mockCtrl   *gomock.Controller
		mockUI     *sharedmocks.MockUI
		mockStager *mocks.MockStager
		mockRunner *mocks.MockRunner
		mockApp    *mocks.MockApp
		mockFS     *mocks.MockFS
		mockHelp   *mocks.MockHelp
		mockConfig *mocks.MockConfig
		cmd        *Run
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = sharedmocks.NewMockUI()
		mockStager = mocks.NewMockStager(mockCtrl)
		mockRunner = mocks.NewMockRunner(mockCtrl)
		mockApp = mocks.NewMockApp(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockHelp = mocks.NewMockHelp(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)
		cmd = &Run{
			UI:     mockUI,
			Stager: mockStager,
			Runner: mockRunner,
			App:    mockApp,
			FS:     mockFS,
			Help:   mockHelp,
			Config: mockConfig,
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
			droplet := newMockBufferCloser(mockCtrl, "some-droplet")
			launcher := newMockBufferCloser(mockCtrl, "some-launcher")
			sshpass := newMockBufferCloser(mockCtrl, "some-sshpass")
			services := service.Services{"some": {{Name: "services"}}}
			forwardedServices := service.Services{"some": {{Name: "forwarded-services"}}}
			forwardConfig := &service.ForwardConfig{
				Host: "some-ssh-host",
			}
			localYML := &local.LocalYML{
				Applications: []*local.AppConfig{
					{Name: "some-other-app"},
					{
						Name:     "some-app",
						Env:      map[string]string{"a": "b"},
						Services: service.Services{"some": {{Name: "overwritten-services"}}},
					},
				},
			}
			gomock.InOrder(
				mockFS.EXPECT().Abs("some-dir").Return("some-abs-dir", nil),
				mockFS.EXPECT().MakeDirAll("some-abs-dir").Return(nil),
				mockFS.EXPECT().IsDirEmpty("some-abs-dir").Return(true, nil),
				mockConfig.EXPECT().Load().Return(localYML, nil),
				mockFS.EXPECT().ReadFile("./some-app.droplet").Return(droplet, int64(100), nil),
				mockStager.EXPECT().Download("/tmp/lifecycle/launcher").Return(local.Stream{launcher, 200}, nil),
				mockApp.EXPECT().Services("some-service-app").Return(services, nil),
				mockApp.EXPECT().Forward("some-forward-app", services).Return(forwardedServices, forwardConfig, nil),
				mockStager.EXPECT().Download("/usr/bin/sshpass").Return(local.Stream{sshpass, 300}, nil),
				mockRunner.EXPECT().Run(&local.RunConfig{
					Droplet:  local.Stream{droplet, 100},
					Launcher: local.Stream{launcher, 200},
					Forwarder: local.Forwarder{
						SSHPass: local.Stream{sshpass, 300},
						Config:  forwardConfig,
					},
					Port:        3000,
					AppDir:      "some-abs-dir",
					AppDirEmpty: true,
					AppConfig: &local.AppConfig{
						Name:     "some-app",
						Env:      map[string]string{"a": "b"},
						Services: forwardedServices,
					},
				}, gomock.Any()).Return(
					int64(0), nil,
				).Do(func(_ *local.RunConfig, c local.Colorizer) {
					Expect(c("some-text")).To(Equal(color.GreenString("some-text")))
				}),
				sshpass.EXPECT().Close(),
				launcher.EXPECT().Close(),
				droplet.EXPECT().Close(),
			)
			Expect(cmd.Run([]string{"run", "some-app", "-p", "3000", "-d", "some-dir", "-s", "some-service-app", "-f", "some-forward-app"})).To(Succeed())
			Expect(mockUI.Out).To(gbytes.Say("Running some-app on port 3000..."))
		})

		// TODO: test app dir when app dir is unspecified (currently tested by integration)
		// TODO: test free port picker when port is unspecified (currently tested by integration)
		// TODO: test different combinations of -s and -f
	})
})
