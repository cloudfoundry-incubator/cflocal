package cmd_test

import (
	"io"
	"io/ioutil"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	. "github.com/sclevine/cflocal/cf/cmd"
	"github.com/sclevine/cflocal/cf/cmd/mocks"
	sharedmocks "github.com/sclevine/cflocal/mocks"
	"github.com/sclevine/forge"
	"github.com/sclevine/forge/app"
)

var _ = Describe("Push", func() {
	var (
		mockCtrl      *gomock.Controller
		mockUI        *sharedmocks.MockUI
		mockRemoteApp *mocks.MockRemoteApp
		mockFS        *mocks.MockFS
		mockHelp      *mocks.MockHelp
		mockConfig    *mocks.MockConfig
		cmd           *Push
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUI = sharedmocks.NewMockUI()
		mockRemoteApp = mocks.NewMockRemoteApp(mockCtrl)
		mockFS = mocks.NewMockFS(mockCtrl)
		mockHelp = mocks.NewMockHelp(mockCtrl)
		mockConfig = mocks.NewMockConfig(mockCtrl)
		cmd = &Push{
			UI:        mockUI,
			RemoteApp: mockRemoteApp,
			FS:        mockFS,
			Help:      mockHelp,
			Config:    mockConfig,
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
			droplet := sharedmocks.NewMockBuffer("some-droplet")
			localYML := &app.LocalYML{
				Applications: []*forge.AppConfig{
					{Name: "some-other-app"},
					{
						Name: "some-app",
						Env:  map[string]string{"some": "env"},
					},
				},
			}
			mockConfig.EXPECT().Load().Return(localYML, nil)
			mockFS.EXPECT().ReadFile("./some-app.droplet").Return(droplet, int64(100), nil)
			gomock.InOrder(
				mockRemoteApp.EXPECT().SetDroplet("some-app", gomock.Any(), int64(100)).Do(func(_ string, r io.Reader, _ int64) {
					Expect(ioutil.ReadAll(r)).To(Equal([]byte("some-droplet")))
				}),
				mockRemoteApp.EXPECT().SetEnv("some-app", map[string]string{"some": "env"}),
				mockRemoteApp.EXPECT().Restart("some-app"),
			)
			Expect(cmd.Run([]string{"push", "some-app", "-e"})).To(Succeed())
			Expect(droplet.Result()).To(BeEmpty())
			Expect(mockUI.Out).To(gbytes.Say("Successfully pushed: some-app"))
		})

		// TODO: test without setting env or restarting
	})
})
