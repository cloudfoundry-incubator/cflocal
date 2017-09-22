package local_test

import (
	"bytes"
	"sort"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
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

var _ = Describe("Runner", func() {
	var (
		runner        *Runner
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

		runner = &Runner{
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

	Describe("#Run", func() {
		It("should run the droplet in a container using the launcher", func() {
			progress := make(chan ui.Progress, 1)
			progress <- mockProgress{Value: "some-progress"}
			close(progress)
			config := &RunConfig{
				Droplet:  engine.NewStream(mockReadCloser{Value: "some-droplet"}, 100),
				Launcher: engine.NewStream(mockReadCloser{Value: "some-launcher"}, 200),
				AppDir:   "some-app-dir",
				RSync:    true,
				Restart:  make(<-chan time.Time),
				Color:    percentColor,
				AppConfig: &AppConfig{
					Name:      "some-app",
					Command:   "some-command",
					Memory:    "512m",
					DiskQuota: "1G",
					StagingEnv: map[string]string{
						"SOME_NA_KEY": "some-na-value",
					},
					RunningEnv: map[string]string{
						"TEST_RUNNING_ENV_KEY": "test-running-env-value",
						"LANG":                 "some-overridden-lang",
					},
					Env: map[string]string{
						"TEST_ENV_KEY": "test-env-value",
						"LANG":         "some-lang",
					},
					Services: service.Services{
						"some-type": {{
							Name: "some-name",
						}},
					},
				},
				NetworkConfig: &NetworkConfig{
					HostIP:   "some-ip",
					HostPort: "400",
				},
			}
			gomock.InOrder(
				mockImage.EXPECT().Pull("cloudfoundry/cflinuxfs2:some-stack-version").Return(progress),
				mockEngine.EXPECT().NewContainer(gomock.Any(), gomock.Any()).Do(func(config *container.Config, hostConfig *container.HostConfig) {
					Expect(config.Hostname).To(Equal("cflocal"))
					Expect(config.User).To(Equal("vcap"))
					Expect(config.ExposedPorts).To(HaveLen(1))
					_, hasPort := config.ExposedPorts["8080/tcp"]
					Expect(hasPort).To(BeTrue())
					sort.Strings(config.Env)
					Expect(config.Env).To(Equal(fixtures.ProvidedRunningEnv("LANG=some-lang")))
					Expect(config.Image).To(Equal("cloudfoundry/cflinuxfs2:some-stack-version"))
					Expect(config.WorkingDir).To(Equal("/home/vcap/app"))
					Expect(config.Entrypoint).To(Equal(strslice.StrSlice{
						"/bin/bash", "-c", fixtures.RunRSyncScript(), "some-command",
					}))
					Expect(hostConfig.Memory).To(Equal(int64(512 * 1024 * 1024)))
					Expect(hostConfig.PortBindings).To(HaveLen(1))
					Expect(hostConfig.PortBindings["8080/tcp"]).To(Equal([]nat.PortBinding{{HostIP: "some-ip", HostPort: "400"}}))
					Expect(hostConfig.NetworkMode).To(BeEmpty())
					Expect(hostConfig.Binds).To(Equal([]string{"some-app-dir:/tmp/local"}))
				}).Return(mockContainer, nil),
			)

			launcherCopy := mockContainer.EXPECT().CopyTo(config.Launcher, "/tmp/lifecycle/launcher")
			dropletCopy := mockContainer.EXPECT().CopyTo(config.Droplet, "/tmp/droplet")

			gomock.InOrder(
				mockContainer.EXPECT().
					Start("[some-app] % ", runner.Logs, config.Restart).Return(int64(100), nil).
					After(launcherCopy).
					After(dropletCopy),
				mockContainer.EXPECT().
					Close(),
			)

			Expect(runner.Run(config)).To(Equal(int64(100)))
			Expect(mockUI.Progress).To(Receive(Equal(mockProgress{Value: "some-progress"})))
		})

		// TODO: test when app dir is empty
		// TODO: test with container networking
	})

	Describe("#Export", func() {
		It("should load the provided droplet into a Docker image with the launcher", func() {
			progress := make(chan ui.Progress, 1)
			progress <- mockProgress{Value: "some-progress"}
			close(progress)
			config := &ExportConfig{
				Droplet:  engine.NewStream(mockReadCloser{Value: "some-droplet"}, 100),
				Launcher: engine.NewStream(mockReadCloser{Value: "some-launcher"}, 200),
				Ref:      "some-ref",
				AppConfig: &AppConfig{
					Name:      "some-app",
					Command:   "some-command",
					Memory:    "512m",
					DiskQuota: "1G",
					StagingEnv: map[string]string{
						"SOME_NA_KEY": "some-na-value",
					},
					RunningEnv: map[string]string{
						"TEST_RUNNING_ENV_KEY": "test-running-env-value",
						"LANG":                 "some-overridden-lang",
					},
					Env: map[string]string{
						"TEST_ENV_KEY": "test-env-value",
						"LANG":         "some-lang",
					},
					Services: service.Services{
						"some-type": {{
							Name: "some-name",
						}},
					},
				},
			}
			gomock.InOrder(
				mockImage.EXPECT().Pull("cloudfoundry/cflinuxfs2:some-stack-version").Return(progress),
				mockEngine.EXPECT().NewContainer(gomock.Any(), gomock.Any()).Do(func(config *container.Config, hostConfig *container.HostConfig) {
					Expect(config.Hostname).To(Equal("cflocal"))
					Expect(config.User).To(Equal("vcap"))
					Expect(config.ExposedPorts).To(HaveLen(1))
					_, hasPort := config.ExposedPorts["8080/tcp"]
					Expect(hasPort).To(BeTrue())
					sort.Strings(config.Env)
					Expect(config.Env).To(Equal(fixtures.ProvidedRunningEnv("LANG=some-lang")))
					Expect(config.Image).To(Equal("cloudfoundry/cflinuxfs2:some-stack-version"))
					Expect(config.Entrypoint).To(Equal(strslice.StrSlice{
						"/bin/bash", "-c", fixtures.CommitScript(), "some-command",
					}))
					Expect(hostConfig).To(BeNil())
				}).Return(mockContainer, nil),
			)

			launcherCopy := mockContainer.EXPECT().CopyTo(config.Launcher, "/tmp/lifecycle/launcher")
			dropletCopy := mockContainer.EXPECT().CopyTo(config.Droplet, "/tmp/droplet")

			gomock.InOrder(
				mockContainer.EXPECT().Commit("some-ref").Return("some-image-id", nil).
					After(launcherCopy).After(dropletCopy),
				mockContainer.EXPECT().Close(),
			)

			Expect(runner.Export(config)).To(Equal("some-image-id"))
			Expect(mockUI.Progress).To(Receive(Equal(mockProgress{Value: "some-progress"})))

		})

		// TODO: test with custom start command
		// TODO: test with empty app dir / without rsync
	})
})
