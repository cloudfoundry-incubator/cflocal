package cmd_test

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/fatih/color"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "code.cloudfoundry.org/cflocal/cf/cmd"
	"code.cloudfoundry.org/cflocal/cf/cmd/mocks"
	sharedmocks "code.cloudfoundry.org/cflocal/mocks"
	"github.com/buildpack/forge"
	"github.com/buildpack/forge/app"
	"github.com/buildpack/forge/engine"
)

var _ = Describe("Stage", func() {
	var (
		mockCtrl      *gomock.Controller
		mockUI        *sharedmocks.MockUI
		mockStager    *mocks.MockStager
		mockRemoteApp *mocks.MockRemoteApp
		mockLocalApp  *mocks.MockLocalApp
		mockFS        *mocks.MockFS
		mockHelp      *mocks.MockHelp
		mockConfig    *mocks.MockConfig
		cmd           *Stage
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = sharedmocks.NewMockUI()
		mockStager = mocks.NewMockStager(mockCtrl)
		mockRemoteApp = mocks.NewMockRemoteApp(mockCtrl)
		mockLocalApp = mocks.NewMockLocalApp(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockHelp = mocks.NewMockHelp(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)
		cmd = &Stage{
			UI:        mockUI,
			Stager:    mockStager,
			RemoteApp: mockRemoteApp,
			TarApp:    mockLocalApp.Tar,
			FS:        mockFS,
			Help:      mockHelp,
			Config:    mockConfig,
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
			appTar := sharedmocks.NewMockBuffer("some-app-tar")
			buildpackZip1 := sharedmocks.NewMockBuffer("some-buildpack-zip-one")
			buildpackZip2 := sharedmocks.NewMockBuffer("some-buildpack-zip-two")
			cache := sharedmocks.NewMockBuffer("some-old-cache")
			droplet := sharedmocks.NewMockBuffer("some-droplet")
			dropletFile := sharedmocks.NewMockBuffer("")

			services := forge.Services{"some": {{Name: "services"}}}
			forwardedServices := forge.Services{"some": {{Name: "forwarded-services"}}}
			forwardConfig := &forge.ForwardDetails{
				Host: "some-ssh-host",
			}

			localYML := &app.YAML{
				Applications: []*forge.AppConfig{
					{
						Name: "some-other-app",
					},
					{
						Name:      "some-app",
						Buildpack: "some-other-buildpack",
						Buildpacks: []string{
							"some-other-buildpack-one",
							"some-other-buildpack-two",
						},
						Env:      map[string]string{"a": "b"},
						Services: forge.Services{"some": {{Name: "overwritten-services"}}},
					},
				},
			}

			mockConfig.EXPECT().Load().Return(localYML, nil)
			mockLocalApp.EXPECT().Tar("some-app-dir").Return(appTar, nil)
			mockFS.EXPECT().ReadFile("some-buildpack-one").Return(buildpackZip1, int64(20), nil)
			mockFS.EXPECT().ReadFile("some-buildpack-two").Return(buildpackZip2, int64(21), nil)
			mockRemoteApp.EXPECT().Services("some-service-app").Return(services, nil)
			mockRemoteApp.EXPECT().Forward("some-forward-app", services).Return(forwardedServices, forwardConfig, nil)
			mockFS.EXPECT().OpenFile("./.some-app.cache").Return(cache, int64(100), nil)
			gomock.InOrder(
				mockStager.EXPECT().Stage(gomock.Any()).Do(
					func(config *forge.StageConfig) {
						Expect(ioutil.ReadAll(config.AppTar)).To(Equal([]byte("some-app-tar")))
						Expect(ioutil.ReadAll(config.Cache)).To(Equal([]byte("some-old-cache")))
						Expect(io.WriteString(config.Cache, "some-new-cache")).To(BeNumerically(">", 0))
						Expect(config.CacheEmpty).To(BeFalse())
						Expect(config.BuildpackZips).To(HaveLen(2))
						buildpackZipOut1 := &bytes.Buffer{}
						Expect(config.BuildpackZips["0135919f1de324456b9f2dbdf1983954"].Out(buildpackZipOut1)).To(Succeed())
						Expect(buildpackZipOut1.String()).To(Equal("some-buildpack-zip-o"))
						buildpackZipOut2 := &bytes.Buffer{}
						Expect(config.BuildpackZips["ab534bf201740a2fa7300aa175acd98c"].Out(buildpackZipOut2)).To(Succeed())
						Expect(buildpackZipOut2.String()).To(Equal("some-buildpack-zip-tw"))
						Expect(config.Stack).To(Equal(BuildStack))
						Expect(config.ForceDetect).To(BeTrue())
						Expect(config.Color("some-text")).To(Equal(color.GreenString("some-text")))
						Expect(config.AppConfig).To(Equal(&forge.AppConfig{
							Name:      "some-app",
							Buildpack: "some-buildpack-two",
							Buildpacks: []string{
								"some-buildpack-one",
								"some-buildpack-two",
							},
							Env:      map[string]string{"a": "b"},
							Services: forwardedServices,
						}))
					},
				).Return(engine.NewStream(droplet, int64(droplet.Len())), nil),
				mockFS.EXPECT().WriteFile("./some-app.droplet").Return(dropletFile, nil),
			)

			Expect(cmd.Run([]string{
				"stage", "some-app", "-e",
				"-b", "some-buildpack-one",
				"-b", "some-buildpack-two",
				"-p", "some-app-dir",
				"-s", "some-service-app",
				"-f", "some-forward-app",
			})).To(Succeed())
			Expect(appTar.Result()).To(BeEmpty())
			Expect(buildpackZip1.Result()).To(Equal("ne"))
			Expect(buildpackZip2.Result()).To(Equal("o"))
			Expect(cache.Result()).To(Equal("some-new-cache"))
			Expect(droplet.Result()).To(BeEmpty())
			Expect(dropletFile.Result()).To(Equal("some-droplet"))
			Expect(mockUI.Out).To(gbytes.Say("Warning: 'some-forward-app' app selected for service forwarding will not be used"))
			Expect(mockUI.Out).To(gbytes.Say("Successfully staged: some-app"))
		})

		// TODO: test buildpack, buildpack zip combinations, and force-detect
		// TODO: test with empty cache
		// TODO: make sure everything is closed in error cases
	})
})
