package cmd_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/sclevine/cflocal/cf/cmd"
	"github.com/sclevine/cflocal/cf/cmd/mocks"
	"github.com/sclevine/cflocal/local"
	sharedmocks "github.com/sclevine/cflocal/mocks"
)

var _ = Describe("Push", func() {
	var (
		mockCtrl   *gomock.Controller
		mockUI     *sharedmocks.MockUI
		mockApp    *mocks.MockApp
		mockFS     *mocks.MockFS
		mockHelp   *mocks.MockHelp
		mockConfig *mocks.MockConfig
		cmd        *Push
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = sharedmocks.NewMockUI()
		mockApp = mocks.NewMockApp(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockHelp = mocks.NewMockHelp(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)
		cmd = &Push{
			UI:     mockUI,
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
		It("should return true when the first argument is push", func() {
			Expect(cmd.Match([]string{"push"})).To(BeTrue())
			Expect(cmd.Match([]string{"not-push"})).To(BeFalse())
			Expect(cmd.Match([]string{})).To(BeFalse())
			Expect(cmd.Match(nil)).To(BeFalse())
		})
	})

	Describe("#Run", func() {
		It("should replace an app's droplet and env vars, then restart it", func() {
			droplet := newMockBufferCloser(mockCtrl, "some-droplet")
			localYML := &local.LocalYML{
				Applications: []*local.AppConfig{
					{Name: "some-other-app"},
					{
						Name: "some-app",
						Env:  map[string]string{"some": "env"},
					},
				},
			}
			gomock.InOrder(
				mockFS.EXPECT().ReadFile("./some-app.droplet").Return(droplet, int64(100), nil),
				mockApp.EXPECT().SetDroplet("some-app", droplet, int64(100)).Return(nil),
				droplet.EXPECT().Close(),
				mockConfig.EXPECT().Load().Return(localYML, nil),
				mockApp.EXPECT().SetEnv("some-app", map[string]string{"some": "env"}).Return(nil),
				mockApp.EXPECT().Restart("some-app").Return(nil),
			)
			Expect(cmd.Run([]string{"push", "some-app", "-e"})).To(Succeed())
			Expect(mockUI.Out).To(gbytes.Say("Successfully pushed: some-app"))
		})

		// TODO: test without setting env or restarting
	})
})
