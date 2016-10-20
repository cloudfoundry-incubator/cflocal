package local_test

import (
	"bytes"
	"errors"

	"github.com/sclevine/cflocal/app"
	. "github.com/sclevine/cflocal/local"
	"github.com/sclevine/cflocal/local/mocks"

	"github.com/fatih/color"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("CF", func() {
	var (
		mockCtrl   *gomock.Controller
		mockUI     *mocks.MockUI
		mockCLI    *mocks.MockCliConnection
		mockStager *mocks.MockStager
		mockRunner *mocks.MockRunner
		mockFS     *mocks.MockFS
		cf         *CF
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = mocks.NewMockUI()
		mockCLI = mocks.NewMockCliConnection(mockCtrl)
		mockStager = mocks.NewMockStager(mockCtrl)
		mockRunner = mocks.NewMockRunner(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		cf = &CF{
			UI:      mockUI,
			Stager:  mockStager,
			Runner:  mockRunner,
			FS:      mockFS,
			CLI:     mockCLI,
			Version: "some-version",
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Run", func() {
		Context("when the subcommand is 'help'", func() {
			It("should run `cf local help`", func() {
				mockCLI.EXPECT().CliCommand("help", "local")
				Expect(cf.Run([]string{"help"})).To(Succeed())
			})

			Context("when printing the help text fails", func() {
				It("should print an error", func() {
					mockCLI.EXPECT().CliCommand("help", "local").Return(nil, errors.New("some error"))
					err := cf.Run([]string{"help"})
					Expect(err).To(MatchError("some error"))
				})
			})
		})

		Context("when the subcommand is '[--]version'", func() {
			It("should output the version", func() {
				Expect(cf.Run([]string{"version"})).To(Succeed())
				Expect(mockUI.Out).To(gbytes.Say("CF Local version some-version"))
				Expect(cf.Run([]string{"--version"})).To(Succeed())
				Expect(mockUI.Out).To(gbytes.Say("CF Local version some-version"))
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
				Expect(cf.Run([]string{"stage", "some-app"})).To(Succeed())
				Expect(file.String()).To(Equal("some-droplet"))
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
				Expect(cf.Run([]string{"run", "some-app"})).To(Succeed())
				Expect(mockUI.Out).To(gbytes.Say("Running some-app..."))
			})
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
