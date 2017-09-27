package local_test

import (
	"bytes"
	"io/ioutil"
	"sort"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/sclevine/cflocal/engine"
	"github.com/sclevine/cflocal/fixtures"
	. "github.com/sclevine/cflocal/local"
	"github.com/sclevine/cflocal/local/mocks"
	sharedmocks "github.com/sclevine/cflocal/mocks"
	"github.com/sclevine/cflocal/service"
	"github.com/sclevine/cflocal/ui"
)

var _ = Describe("Stager", func() {
	var (
		stager        *Stager
		mockCtrl      *gomock.Controller
		mockUI        *sharedmocks.MockUI
		mockEngine    *mocks.MockEngine
		mockImage     *mocks.MockImage
		mockVersioner *mocks.MockVersioner
		mockContainer *mocks.MockContainer
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = sharedmocks.NewMockUI()
		mockEngine = mocks.NewMockEngine(mockCtrl)
		mockImage = mocks.NewMockImage(mockCtrl)
		mockVersioner = mocks.NewMockVersioner(mockCtrl)
		mockContainer = mocks.NewMockContainer(mockCtrl)

		stager = &Stager{
			DiegoVersion: "some-diego-version",
			GoVersion:    "some-go-version",
			StackVersion: "some-stack-version",
			Logs:         bytes.NewBufferString("some-logs"),
			UI:           mockUI,
			Engine:       mockEngine,
			Image:        mockImage,
			Versioner:    mockVersioner,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Stage", func() {
		It("should return a droplet of a staged app", func() {
			localCache := sharedmocks.NewMockBuffer("some-old-cache")
			remoteCache := sharedmocks.NewMockBuffer("some-new-cache")
			remoteCacheStream := engine.NewStream(remoteCache, int64(remoteCache.Len()))
			dropletStream := engine.NewStream(mockReadCloser{Value: "some-droplet"}, 100)

			progress := make(chan ui.Progress, 1)
			progress <- mockProgress{Value: "some-progress"}
			close(progress)

			config := &StageConfig{
				AppTar:     bytes.NewBufferString("some-app-tar"),
				Cache:      localCache,
				CacheEmpty: false,
				AppDir:     "some-app-dir",
				RSync:      true,
				Color:      percentColor,
				AppConfig: &AppConfig{
					Name:      "some-app",
					Buildpack: "some-buildpack",
					Buildpacks: []string{
						"some-buildpack-one",
						"some-buildpack-two",
					},
					StagingEnv: map[string]string{
						"TEST_STAGING_ENV_KEY": "test-staging-env-value",
						"MEMORY_LIMIT":         "256m",
					},
					RunningEnv: map[string]string{
						"SOME_NA_KEY": "some-na-value",
					},
					Env: map[string]string{
						"TEST_ENV_KEY": "test-env-value",
						"MEMORY_LIMIT": "1024m",
					},
					Services: service.Services{
						"some-type": {{
							Name: "some-name",
						}},
					},
				},
			}

			Expect(Buildpacks).NotTo(HaveLen(0))
			for _, buildpack := range Buildpacks {
				mockVersioner.EXPECT().Build(buildpack.URL, buildpack.VersionURL).Return(buildpack.Name+"-versioned-url", nil)
			}

			gomock.InOrder(
				mockImage.EXPECT().Build(gomock.Any(), gomock.Any()).Do(func(tag string, dockerfile engine.Stream) {
					Expect(tag).To(Equal("cflocal"))
					dfBytes, err := ioutil.ReadAll(dockerfile)
					Expect(err).NotTo(HaveOccurred())

					Expect(dockerfile.Size).To(Equal(int64(len(dfBytes))))
					Expect(dfBytes).To(ContainSubstring("FROM cloudfoundry/cflinuxfs2:some-stack-version"))
					Expect(dfBytes).To(ContainSubstring("gosome-go-version.linux-amd64"))
					Expect(dfBytes).To(ContainSubstring(`git checkout "vsome-diego-version"`))
					Expect(dfBytes).To(ContainSubstring(`"go_buildpack-versioned-url"`))
					Expect(dfBytes).To(ContainSubstring("/tmp/buildpacks/d222e8f339cb0c77b7a3051618bf9ca7"))
				}).Return(progress),
				mockEngine.EXPECT().NewContainer(gomock.Any(), gomock.Any()).Do(func(config *container.Config, hostConfig *container.HostConfig) {
					Expect(config.Hostname).To(Equal("cflocal"))
					Expect(config.User).To(Equal("root"))
					Expect(config.ExposedPorts).To(HaveLen(0))
					sort.Strings(config.Env)
					Expect(config.Env).To(Equal(fixtures.ProvidedStagingEnv("MEMORY_LIMIT=1024m")))
					Expect(config.Image).To(Equal("cflocal"))
					Expect(config.WorkingDir).To(Equal("/home/vcap"))
					Expect(config.Entrypoint).To(Equal(strslice.StrSlice{
						"/bin/bash", "-c", fixtures.StageRSyncScript(),
						"some-buildpack-one,some-buildpack-two", "true",
					}))
					Expect(hostConfig.Binds).To(Equal([]string{"some-app-dir:/tmp/local"}))
				}).Return(mockContainer, nil),
			)

			gomock.InOrder(
				mockContainer.EXPECT().ExtractTo(config.AppTar, "/tmp/app"),
				mockContainer.EXPECT().ExtractTo(localCache, "/tmp/cache"),
				mockContainer.EXPECT().Start("[some-app] % ", stager.Logs, nil).Return(int64(0), nil),
				mockContainer.EXPECT().CopyFrom("/tmp/output-cache").Return(remoteCacheStream, nil),
				mockContainer.EXPECT().CopyFrom("/tmp/droplet").Return(dropletStream, nil),
				mockContainer.EXPECT().CloseAfterStream(&dropletStream),
			)

			Expect(stager.Stage(config)).To(Equal(dropletStream))
			Expect(localCache.Close()).To(Succeed())
			Expect(localCache.Result()).To(Equal("some-new-cache"))
			Expect(remoteCache.Result()).To(BeEmpty())
			Expect(mockUI.Out).To(gbytes.Say("Buildpacks: some-buildpack-one, some-buildpack-two"))
			Expect(mockUI.Progress).To(Receive(Equal(mockProgress{Value: "some-progress"})))
		})

		// TODO: test unavailable buildpack versions
		// TODO: test single-buildpack case
		// TODO: test non-zero command return status
		// TODO: test no app dir case
		// TODO: test without rsync
	})

	Describe("#Download", func() {
		It("should return the specified file", func() {
			progress := make(chan ui.Progress, 1)
			progress <- mockProgress{Value: "some-progress"}
			close(progress)

			Expect(Buildpacks).NotTo(HaveLen(0))
			for _, buildpack := range Buildpacks {
				mockVersioner.EXPECT().Build(buildpack.URL, buildpack.VersionURL).Return(buildpack.Name+"-versioned-url", nil)
			}

			gomock.InOrder(
				mockImage.EXPECT().Build(gomock.Any(), gomock.Any()).Do(func(tag string, dockerfile engine.Stream) {
					Expect(tag).To(Equal("cflocal"))
					dfBytes, err := ioutil.ReadAll(dockerfile)
					Expect(err).NotTo(HaveOccurred())

					Expect(dockerfile.Size).To(Equal(int64(len(dfBytes))))
					Expect(dfBytes).To(ContainSubstring("FROM cloudfoundry/cflinuxfs2:some-stack-version"))
					Expect(dfBytes).To(ContainSubstring("gosome-go-version.linux-amd64"))
					Expect(dfBytes).To(ContainSubstring(`git checkout "vsome-diego-version"`))
					Expect(dfBytes).To(ContainSubstring(`"go_buildpack-versioned-url"`))
					Expect(dfBytes).To(ContainSubstring("/tmp/buildpacks/d222e8f339cb0c77b7a3051618bf9ca7"))
				}).Return(progress),
				mockEngine.EXPECT().NewContainer(gomock.Any(), gomock.Any()).Do(func(config *container.Config, hostConfig *container.HostConfig) {
					Expect(config.Hostname).To(Equal("cflocal"))
					Expect(config.User).To(Equal("root"))
					Expect(config.ExposedPorts).To(HaveLen(0))
					Expect(config.Image).To(Equal("cflocal"))
					Expect(config.Entrypoint).To(Equal(strslice.StrSlice{"read"}))
					Expect(hostConfig).To(BeNil())
				}).Return(mockContainer, nil),
			)

			stream := engine.NewStream(mockReadCloser{Value: "some-stream"}, 100)
			gomock.InOrder(
				mockContainer.EXPECT().CopyFrom("/some-path").Return(stream, nil),
				mockContainer.EXPECT().CloseAfterStream(&stream),
			)

			Expect(stager.Download("/some-path")).To(Equal(stream))
			Expect(mockUI.Progress).To(Receive(Equal(mockProgress{Value: "some-progress"})))
		})
	})
})
