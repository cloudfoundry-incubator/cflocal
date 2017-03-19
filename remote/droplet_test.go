package remote_test

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sclevine/cflocal/mocks"
	. "github.com/sclevine/cflocal/remote"
	"github.com/sclevine/cflocal/testutil"
)

var _ = Describe("App - Droplet", func() {
	var (
		mockCtrl *gomock.Controller
		mockCLI  *mocks.MockCliConnection
		mockUI   *mocks.MockUI
		server   *testutil.Server
		app      *App
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCLI = mocks.NewMockCliConnection(mockCtrl)
		mockUI = mocks.NewMockUI()
		server = testutil.Serve(mockCLI)
		app = &App{CLI: mockCLI, UI: mockUI}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Droplet", func() {
		It("should return the app's droplet", func() {
			req, _ := server.HandleApp("some-name", http.StatusOK, "some-droplet")

			droplet, size, err := app.Droplet("some-name")
			Expect(err).NotTo(HaveOccurred())
			defer droplet.Close()

			Expect(size).To(Equal(int64(12)))
			Expect(ioutil.ReadAll(droplet)).To(Equal([]byte("some-droplet")))

			Expect(req.Method).To(Equal("GET"))
			Expect(req.Path).To(Equal("/v2/apps/some-app-guid/droplet/download"))
			Expect(req.Authenticated).To(BeTrue())
		})
	})

	Describe("#SetDroplet", func() {
		It("should upload the app's droplet", func() {
			server := testutil.Serve(mockCLI)
			appReq, appCalls := server.HandleApp("some-name", http.StatusCreated, `{"entity": {"guid": "some-guid", "status": "queued"}}`)
			jobReq1, jobCalls1 := server.Handle(true, http.StatusOK, `{"entity": {"guid": "some-guid", "status": "running"}}`)
			jobReq2, jobCalls2 := server.Handle(true, http.StatusOK, `{"entity": {"guid": "some-guid", "status": "finished"}}`)

			appCalls.Before(jobCalls1.Before(jobCalls2))

			droplet := bytes.NewBufferString("some-droplet")
			Expect(app.SetDroplet("some-name", droplet, int64(droplet.Len()))).To(Succeed())

			Expect(appReq.Method).To(Equal("PUT"))
			Expect(appReq.Path).To(Equal("/v2/apps/some-app-guid/droplet/upload"))
			Expect(appReq.Authenticated).To(BeTrue())
			Expect(appReq.ContentType).To(MatchRegexp("multipart/form-data; boundary=[A-Fa-f0-9]{60}"))
			Expect(appReq.ContentLength).To(Equal(int64(264)))
			boundary := appReq.ContentType[len(appReq.ContentType)-60:]
			Expect(appReq.Body).To(MatchRegexp(
				`--%s\s+Content-Disposition: form-data; name="droplet"; filename="some-name\.droplet"\s+`+
					`Content-Type: application/octet-stream\s+some-droplet\s+--%[1]s--`, boundary,
			))

			Expect(jobReq1.Method).To(Equal("GET"))
			Expect(jobReq1.Path).To(Equal("/v2/jobs/some-guid"))
			Expect(jobReq1.Authenticated).To(BeTrue())

			Expect(jobReq2.Method).To(Equal("GET"))
			Expect(jobReq2.Path).To(Equal("/v2/jobs/some-guid"))
			Expect(jobReq2.Authenticated).To(BeTrue())
		})
	})
})
