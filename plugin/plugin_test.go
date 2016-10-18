package plugin_test

import (
	"bytes"
	"errors"

	"github.com/sclevine/cflocal/app"
	. "github.com/sclevine/cflocal/plugin"
	"github.com/sclevine/cflocal/plugin/mocks"

	cfplugin "code.cloudfoundry.org/cli/plugin"
	"github.com/fatih/color"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Plugin", func() {
	var (
		mockCtrl   *gomock.Controller
		mockUI     *mocks.MockUI
		mockCLI    *mocks.MockCliConnection
		mockStager *mocks.MockStager
		mockRunner *mocks.MockRunner
		mockFS     *mocks.MockFS
		plugin     *Plugin
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI()
		mockCLI = mocks.NewMockCliConnection(mockCtrl)
		mockStager = mocks.NewMockStager(mockCtrl)
		mockRunner = mocks.NewMockRunner(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		plugin = &Plugin{
			UI:     mockUI,
			Stager: mockStager,
			Runner: mockRunner,
			FS:     mockFS,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Run", func() {
		Context("when the subcommand is 'help'", func() {
			It("should run `cf local help`", func() {
				mockCLI.EXPECT().CliCommand("help", "local")
				plugin.Run(mockCLI, []string{"dev", "help"})
			})

			Context("when printing the help text fails", func() {
				It("should print an error", func() {
					mockCLI.EXPECT().CliCommand("help", "local").Return(nil, errors.New("some error"))
					plugin.Run(mockCLI, []string{"local", "help"})
					Expect(mockUI.Err).To(MatchError("some error"))
				})
			})
		})

		Context("when the subcommand is '[--]version'", func() {
			It("should output the version", func() {
				plugin.Version = cfplugin.VersionType{100, 200, 300}
				plugin.Run(mockCLI, []string{"local", "version"})
				Expect(mockUI.Out).To(gbytes.Say("CF Local version 100.200.300"))
				plugin.Run(mockCLI, []string{"local", "--version"})
				Expect(mockUI.Out).To(gbytes.Say("CF Local version 100.200.300"))
			})
		})

		Context("when the subcommand is 'stage'", func() {
			It("should build a droplet", func() {
				appTar := newMockBufferCloser(mockCtrl)
				droplet := newMockBufferCloser(mockCtrl, "some-droplet")
				file := newMockBufferCloser(mockCtrl)
				gomock.InOrder(
					mockFS.EXPECT().Tar(".").Return(appTar, nil),
					mockStager.EXPECT().Stage("some-app", gomock.Any(), &app.StageConfig{
						AppTar:     appTar,
						Buildpacks: Buildpacks,
					}).Return(
						droplet, int64(100), nil,
					).Do(func(_ string, c app.Colorizer, _ *app.StageConfig) {
						Expect(c("some-text")).To(Equal(color.GreenString("some-text")))
					}),
					mockFS.EXPECT().WriteFile("./some-app.droplet").Return(file, nil),
					file.EXPECT().Close(),
					droplet.EXPECT().Close(),
					appTar.EXPECT().Close(),
				)
				plugin.Run(mockCLI, []string{"local", "stage", "some-app"})
				Expect(file.String()).To(Equal("some-droplet"))
				Expect(mockUI.Err).NotTo(HaveOccurred())
				Expect(mockUI.Out).To(gbytes.Say("Staging of some-app successful."))
			})
		})

		Context("when the subcommand is 'run'", func() {
			It("should build a droplet", func() {
				droplet := newMockBufferCloser(mockCtrl)
				launcher := newMockBufferCloser(mockCtrl)
				gomock.InOrder(
					mockFS.EXPECT().ReadFile("./some-app.droplet").Return(droplet, int64(100), nil),
					mockStager.EXPECT().Launcher().Return(launcher, int64(200), nil),
					mockRunner.EXPECT().Run("some-app", gomock.Any(), &app.RunConfig{
						Droplet:      droplet,
						DropletSize:  int64(100),
						Launcher:     launcher,
						LauncherSize: int64(200),
						Port:         3000,
					}).Return(
						0, nil,
					).Do(func(_ string, c app.Colorizer, _ *app.RunConfig) {
						Expect(c("some-text")).To(Equal(color.GreenString("some-text")))
					}),
					launcher.EXPECT().Close(),
					droplet.EXPECT().Close(),
				)
				plugin.Run(mockCLI, []string{"local", "run", "some-app"})
				Expect(mockUI.Err).NotTo(HaveOccurred())
				Expect(mockUI.Out).To(gbytes.Say("Running some-app..."))
			})
		})

		Context("when uninstalling", func() {
			It("should return immediately", func() {
				plugin.Run(mockCLI, []string{"CLI-MESSAGE-UNINSTALL"})
				Expect(len(mockUI.Out.Contents())).To(BeZero())
				Expect(mockUI.Err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("#GetMetadata", func() {
		It("should populate the version field", func() {
			plugin.Version = cfplugin.VersionType{100, 200, 300}
			Expect(plugin.GetMetadata().Version).To(Equal(cfplugin.VersionType{100, 200, 300}))
		})
	})
})

type mockBufferCloser struct {
	*mocks.MockCloser
	*bytes.Buffer
}

func newMockBufferCloser(ctrl *gomock.Controller, contents ...string) *mockBufferCloser {
	bc := &mockBufferCloser{mocks.NewMockCloser(ctrl), &bytes.Buffer{}}
	for _, v := range contents {
		bc.Buffer.Write([]byte(v))
	}
	return bc
}
