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
		mockContainer *mocks.MockContainer
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = sharedmocks.NewMockUI()
		mockEngine = mocks.NewMockEngine(mockCtrl)
		mockImage = mocks.NewMockImage(mockCtrl)
		mockContainer = mocks.NewMockContainer(mockCtrl)

		stager = &Stager{
			DiegoVersion: "some-diego-version",
			GoVersion:    "some-go-version",
			StackVersion: "some-stack-version",
			Logs:         bytes.NewBufferString("some-logs"),
			UI:           mockUI,
			Engine:       mockEngine,
			Image:        mockImage,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Stage", func() {
		It("should return a droplet of a staged app", func() {
			progress := make(chan ui.Progress, 1)
			progress <- mockProgress{Value: "some-progress"}
			close(progress)

			config := &StageConfig{
				AppTar:     bytes.NewBufferString("some-app-tar"),
				Buildpacks: []string{"some-first-buildpack", "some-second-buildpack"},
				AppConfig: &AppConfig{
					Name: "some-app",
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

			gomock.InOrder(
				mockImage.EXPECT().Build(gomock.Any(), gomock.Any()).Do(func(tag string, dockerfile engine.Stream) {
					Expect(tag).To(Equal("cflocal"))
					dfBytes, err := ioutil.ReadAll(dockerfile)
					Expect(err).NotTo(HaveOccurred())

					Expect(dockerfile.Size).To(Equal(int64(len(dfBytes))))
					Expect(dfBytes).To(ContainSubstring("FROM cloudfoundry/cflinuxfs2:some-stack-version"))
					Expect(dfBytes).To(ContainSubstring("gosome-go-version.linux-amd64"))
					Expect(dfBytes).To(ContainSubstring(`git checkout "vsome-diego-version"`))
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
						"/bin/bash", "-c", StagerScript,
						"some-first-buildpack,some-second-buildpack", "false",
					}))
					Expect(hostConfig).To(BeNil())
				}).Return(mockContainer, nil),
			)

			droplet := engine.NewStream(mockReadCloser{Value: "some-droplet"}, 100)
			gomock.InOrder(
				mockContainer.EXPECT().ExtractTo(config.AppTar, "/tmp/app"),
				mockContainer.EXPECT().Start("[some-app] % ", stager.Logs).Return(int64(0), nil),
				mockContainer.EXPECT().CopyFrom("/tmp/droplet").Return(droplet, nil),
				mockContainer.EXPECT().CloseAfterStream(&droplet),
			)

			Expect(stager.Stage(config, percentColor)).To(Equal(droplet))
			Expect(mockUI.Progress).To(Receive(Equal(mockProgress{Value: "some-progress"})))
		})

		// TODO: test single-buildpack case
		// TODO: test non-zero command return status
	})

	Describe("#Download", func() {
		It("should return the specified file", func() {
			progress := make(chan ui.Progress, 1)
			progress <- mockProgress{Value: "some-progress"}
			close(progress)

			gomock.InOrder(
				mockImage.EXPECT().Build(gomock.Any(), gomock.Any()).Do(func(tag string, dockerfile engine.Stream) {
					Expect(tag).To(Equal("cflocal"))
					dfBytes, err := ioutil.ReadAll(dockerfile)
					Expect(err).NotTo(HaveOccurred())

					Expect(dockerfile.Size).To(Equal(int64(len(dfBytes))))
					Expect(dfBytes).To(ContainSubstring("FROM cloudfoundry/cflinuxfs2:some-stack-version"))
					Expect(dfBytes).To(ContainSubstring("gosome-go-version.linux-amd64"))
					Expect(dfBytes).To(ContainSubstring(`git checkout "vsome-diego-version"`))
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
