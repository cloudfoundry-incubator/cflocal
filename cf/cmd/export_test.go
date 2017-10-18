package cmd_test

import (
	"io/ioutil"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/sclevine/cflocal/cf/cmd"
	"github.com/sclevine/cflocal/cf/cmd/mocks"
	"github.com/sclevine/forge/engine"
	"github.com/sclevine/forge"
	sharedmocks "github.com/sclevine/cflocal/mocks"
	"github.com/sclevine/cflocal/service"
)

var _ = Describe("Export", func() {
	var (
		mockCtrl   *gomock.Controller
		mockUI     *sharedmocks.MockUI
		mockStager *mocks.MockStager
		mockRunner *mocks.MockRunner
		mockFS     *mocks.MockFS
		mockHelp   *mocks.MockHelp
		mockConfig *mocks.MockConfig
		cmd        *Export
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = sharedmocks.NewMockUI()
		mockStager = mocks.NewMockStager(mockCtrl)
		mockRunner = mocks.NewMockRunner(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockHelp = mocks.NewMockHelp(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)
		cmd = &Export{
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
		It("should return true when the first argument is export", func() {
			Expect(cmd.Match([]string{"export"})).To(BeTrue())
			Expect(cmd.Match([]string{"not-export"})).To(BeFalse())
			Expect(cmd.Match([]string{})).To(BeFalse())
			Expect(cmd.Match(nil)).To(BeFalse())
		})
	})

	Describe("#Run", func() {
		It("should export a droplet as a Docker image", func() {
			droplet := sharedmocks.NewMockBuffer("some-droplet")
			launcher := sharedmocks.NewMockBuffer("some-launcher")
			localYML := &forge.LocalYML{
				Applications: []*forge.AppConfig{
					{Name: "some-other-app"},
					{
						Name:     "some-app",
						Env:      map[string]string{"a": "b"},
						Services: service.Services{"some": {{Name: "services"}}},
					},
				},
			}
			mockConfig.EXPECT().Load().Return(localYML, nil)
			mockFS.EXPECT().ReadFile("./some-app.droplet").Return(droplet, int64(100), nil)
			mockStager.EXPECT().Download("/tmp/lifecycle/launcher").Return(engine.NewStream(launcher, 200), nil)
			mockRunner.EXPECT().Export(gomock.Any()).Do(
				func(config *forge.ExportConfig) {
					Expect(ioutil.ReadAll(config.Droplet)).To(Equal([]byte("some-droplet")))
					Expect(ioutil.ReadAll(config.Launcher)).To(Equal([]byte("some-launcher")))
					Expect(config.Ref).To(Equal("some-reference"))
					Expect(config.AppConfig).To(Equal(&forge.AppConfig{
						Name:     "some-app",
						Env:      map[string]string{"a": "b"},
						Services: service.Services{"some": {{Name: "services"}}},
					}))
				},
			).Return("some-id", nil)

			Expect(cmd.Run([]string{"export", "some-app", "-r", "some-reference"})).To(Succeed())
			Expect(droplet.Result()).To(BeEmpty())
			Expect(launcher.Result()).To(BeEmpty())
			Expect(mockUI.Out).To(gbytes.Say("Exported some-app as some-reference with ID: some-id"))
		})

		// TODO: test without reference
	})
})
