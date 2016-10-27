package cf_test

import (
	"bytes"
	"errors"

	. "github.com/sclevine/cflocal/cf"
	"github.com/sclevine/cflocal/cf/mocks"
	"github.com/sclevine/cflocal/local"
	sharedmocks "github.com/sclevine/cflocal/mocks"
	"github.com/sclevine/cflocal/remote"

	"github.com/fatih/color"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("CF", func() {
	var (
		mockCtrl   *gomock.Controller
		mockUI     *sharedmocks.MockUI
		mockStager *mocks.MockStager
		mockRunner *mocks.MockRunner
		mockApp    *mocks.MockApp
		mockFS     *mocks.MockFS
		mockHelp   *mocks.MockHelp
		mockConfig *mocks.MockConfig
		cf         *CF
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
		cf = &CF{
			UI:      mockUI,
			Stager:  mockStager,
			Runner:  mockRunner,
			App:     mockApp,
			FS:      mockFS,
			Help:    mockHelp,
			Config:  mockConfig,
			Version: "some-version",
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Run", func() {
		Context("when the subcommand is 'help'", func() {
			It("should show the help text", func() {
				mockHelp.EXPECT().Show()
				Expect(cf.Run([]string{"help"})).To(Succeed())
			})

			Context("when printing the help text fails", func() {
				It("should print an error", func() {
					mockHelp.EXPECT().Show().Return(errors.New("some error"))
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
					mockFS.EXPECT().Tar(".").Return(appTar, nil),
					mockConfig.EXPECT().Load().Return(localYML, nil),
					mockStager.EXPECT().Stage(&local.StageConfig{
						AppTar:     appTar,
						Buildpacks: []string{"some-buildpack"},
						AppConfig: &local.AppConfig{
							Name: "some-app",
							Env:  map[string]string{"a": "b"},
						},
					}, gomock.Any()).Return(
						droplet, int64(100), nil,
					).Do(func(_ *local.StageConfig, c local.Colorizer) {
						Expect(c("some-text")).To(Equal(color.GreenString("some-text")))
					}),
					mockFS.EXPECT().WriteFile("./some-app.droplet").Return(file, nil),
					file.EXPECT().Close(),
					droplet.EXPECT().Close(),
					appTar.EXPECT().Close(),
				)
				Expect(cf.Run([]string{"stage", "-b", "some-buildpack", "some-app"})).To(Succeed())
				Expect(file.String()).To(Equal("some-droplet"))
				Expect(mockUI.Out).To(gbytes.Say("Successfully staged: some-app"))
			})

			// test not providing a buildpack
		})

		Context("when the subcommand is 'run'", func() {
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
					mockFS.EXPECT().ReadFile("./some-app.droplet").Return(droplet, int64(100), nil),
					mockStager.EXPECT().Launcher().Return(launcher, int64(200), nil),
					mockConfig.EXPECT().Load().Return(localYML, nil),
					mockRunner.EXPECT().Run(&local.RunConfig{
						Droplet:      droplet,
						DropletSize:  int64(100),
						Launcher:     launcher,
						LauncherSize: int64(200),
						Port:         3000,
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
				Expect(cf.Run([]string{"run", "-p", "3000", "some-app"})).To(Succeed())
				Expect(mockUI.Out).To(gbytes.Say("Running some-app on port 3000..."))
			})

			// test free port picker when port is unspecified (currently tested by integration)
		})

		Context("when the subcommand is 'pull'", func() {
			It("should download a droplet and save its env vars", func() {
				droplet := newMockBufferCloser(mockCtrl, "some-droplet")
				file := newMockBufferCloser(mockCtrl)
				env := &remote.AppEnv{
					Staging: map[string]string{"a": "b"},
					Running: map[string]string{"c": "d"},
					App:     map[string]string{"e": "f"},
				}
				oldLocalYML := &local.LocalYML{
					Applications: []*local.AppConfig{
						{Name: "some-other-app"},
						{
							Name:       "some-app",
							Command:    "some-old-command",
							StagingEnv: map[string]string{"g": "h"},
							RunningEnv: map[string]string{"i": "j"},
							Env:        map[string]string{"k": "l"},
						},
					},
				}
				newLocalYML := &local.LocalYML{
					Applications: []*local.AppConfig{
						{Name: "some-other-app"},
						{
							Name:       "some-app",
							Command:    "some-command",
							StagingEnv: map[string]string{"a": "b"},
							RunningEnv: map[string]string{"c": "d"},
							Env:        map[string]string{"e": "f"},
						},
					},
				}
				gomock.InOrder(
					mockApp.EXPECT().Droplet("some-app").Return(droplet, int64(100), nil),
					mockFS.EXPECT().WriteFile("./some-app.droplet").Return(file, nil),
					file.EXPECT().Close(),
					droplet.EXPECT().Close(),
					mockConfig.EXPECT().Load().Return(oldLocalYML, nil),
					mockApp.EXPECT().Env("some-app").Return(env, nil),
					mockApp.EXPECT().Command("some-app").Return("some-command", nil),
					mockConfig.EXPECT().Save(newLocalYML).Return(nil),
				)
				Expect(cf.Run([]string{"pull", "some-app"})).To(Succeed())
				Expect(file.String()).To(Equal("some-droplet"))
				Expect(mockUI.Out).To(gbytes.Say("Successfully downloaded: some-app"))
			})

			// test when app isn't in local.yml
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
