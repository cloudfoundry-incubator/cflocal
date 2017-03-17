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

var _ = Describe("Stage", func() {
	var (
		mockCtrl   *gomock.Controller
		mockUI     *sharedmocks.MockUI
		mockStager *mocks.MockStager
		mockApp    *mocks.MockApp
		mockFS     *mocks.MockFS
		mockHelp   *mocks.MockHelp
		mockConfig *mocks.MockConfig
		cmd        *Stage
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = sharedmocks.NewMockUI()
		mockStager = mocks.NewMockStager(mockCtrl)
		mockApp = mocks.NewMockApp(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockHelp = mocks.NewMockHelp(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)
		cmd = &Stage{
			UI:     mockUI,
			Stager: mockStager,
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
		It("should return true when the first argument is stage", func() {
			Expect(cmd.Match([]string{"stage"})).To(BeTrue())
			Expect(cmd.Match([]string{"not-stage"})).To(BeFalse())
			Expect(cmd.Match([]string{})).To(BeFalse())
			Expect(cmd.Match(nil)).To(BeFalse())
		})
	})

	Describe("#Run", func() {
		It("should build a droplet", func() {
			appTar := newMockBufferCloser(mockCtrl)
			droplet := newMockBufferCloser(mockCtrl, "some-droplet")
			file := newMockBufferCloser(mockCtrl)
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
				mockConfig.EXPECT().Load().Return(localYML, nil),
				mockFS.EXPECT().Tar(".").Return(appTar, nil),
				mockApp.EXPECT().Services("some-service-app").Return(services, nil),
				mockApp.EXPECT().Forward("some-forward-app", services).Return(forwardedServices, forwardConfig, nil),
				mockStager.EXPECT().Stage(&local.StageConfig{
					AppTar:     appTar,
					Buildpacks: []string{"some-buildpack"},
					AppConfig: &local.AppConfig{
						Name:     "some-app",
						Env:      map[string]string{"a": "b"},
						Services: forwardedServices,
					},
				}, gomock.Any()).Return(
					local.Stream{droplet, 100}, nil,
				).Do(func(_ *local.StageConfig, c local.Colorizer) {
					Expect(c("some-text")).To(Equal(color.GreenString("some-text")))
				}),
				mockFS.EXPECT().WriteFile("./some-app.droplet").Return(file, nil),
				file.EXPECT().Close(),
				droplet.EXPECT().Close(),
				appTar.EXPECT().Close(),
			)
			Expect(cmd.Run([]string{"stage", "some-app", "-b", "some-buildpack", "-s", "some-service-app", "-f", "some-forward-app"})).To(Succeed())
			Expect(file.String()).To(Equal("some-droplet"))
			Expect(mockUI.Out).To(gbytes.Say("Warning: 'some-forward-app' app selected for service forwarding will not be used"))
			Expect(mockUI.Out).To(gbytes.Say("Downloading some-buildpack..."))
			Expect(mockUI.Out).To(gbytes.Say("Successfully staged: some-app"))
		})

		// TODO: test not providing a buildpack
	})
})
