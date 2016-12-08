package cmd_test

import (
	. "github.com/sclevine/cflocal/cf/cmd"
	"github.com/sclevine/cflocal/cf/cmd/mocks"
	"github.com/sclevine/cflocal/local"
	sharedmocks "github.com/sclevine/cflocal/mocks"

	"github.com/fatih/color"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Run", func() {
	var (
		mockCtrl   *gomock.Controller
		mockUI     *sharedmocks.MockUI
		mockStager *mocks.MockStager
		mockRunner *mocks.MockRunner
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
		mockFS = mocks.NewMockFS(mockCtrl)
		mockHelp = mocks.NewMockHelp(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)
		cmd = &Run{
			UI:     mockUI,
			Stager: mockStager,
			Runner: mockRunner,
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
			droplet := newMockBufferCloser(mockCtrl)
			launcher := newMockBufferCloser(mockCtrl)
			localYML := &local.LocalYML{
				Applications: []*local.AppConfig{
					{Name: "some-other-app"},
					{
						Name: "some-app",
						Env:  map[string]string{"a": "b"},
					},
				},
			}
			gomock.InOrder(
				mockFS.EXPECT().Abs("some-dir").Return("some-abs-dir", nil),
				mockFS.EXPECT().MakeDirAll("some-abs-dir").Return(nil),
				mockFS.EXPECT().IsDirEmpty("some-abs-dir").Return(true, nil),
				mockFS.EXPECT().ReadFile("./some-app.droplet").Return(droplet, int64(100), nil),
				mockStager.EXPECT().Launcher().Return(launcher, int64(200), nil),
				mockConfig.EXPECT().Load().Return(localYML, nil),
				mockRunner.EXPECT().Run(&local.RunConfig{
					Droplet:      droplet,
					DropletSize:  int64(100),
					Launcher:     launcher,
					LauncherSize: int64(200),
					Port:         3000,
					AppDir:       "some-abs-dir",
					AppDirEmpty:  true,
					AppConfig: &local.AppConfig{
						Name: "some-app",
						Env:  map[string]string{"a": "b"},
					},
				}, gomock.Any()).Return(
					0, nil,
				).Do(func(_ *local.RunConfig, c local.Colorizer) {
					Expect(c("some-text")).To(Equal(color.GreenString("some-text")))
				}),
				launcher.EXPECT().Close(),
				droplet.EXPECT().Close(),
			)
			Expect(cmd.Run([]string{"run", "-p", "3000", "-d", "some-dir", "some-app"})).To(Succeed())
			Expect(mockUI.Out).To(gbytes.Say("Running some-app on port 3000..."))
		})

		// test app dir when app dir is unspecified (currently tested by integration)
		// test free port picker when port is unspecified (currently tested by integration)
	})
})
