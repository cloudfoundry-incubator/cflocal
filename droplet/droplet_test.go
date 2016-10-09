package droplet_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	docker "github.com/docker/docker/client"
	. "github.com/sclevine/cflocal/droplet"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

// fixture.tgz: 0644 0 0 14 2016-10-05 23:03 some-file -> some-contents
// hexdump -v -e '"\\\x" 1/1 "%02x"' fixture.tgz
const tgzFixture = "" +
	"\x1f\x8b\x08\x00\xbf\xc0\xf5\x57\x00\x03\xed\xd1\x31\x0a\xc2\x40\x10\x46" +
	"\xe1\xa9\x3d\xc5\x5e\x40\x98\x31\x9b\xec\x79\x44\x46\x14\x62\x16\xb2\x2b" +
	"\x5e\xdf\x68\x91\x22\x85\xd8\x24\x69\xde\xd7\xfc\xcd\x14\x0f\xa6\xe4\x87" +
	"\x1f\xaf\xf7\xde\x65\x3d\x3a\xe9\x62\xfc\xee\x64\xb9\xaa\xd6\x89\x9d\x52" +
	"\x6a\x9b\x26\x45\x4d\xa2\x66\xad\x45\x09\xba\x62\xd3\xec\x59\xea\x79\x0c" +
	"\x41\xc6\x9c\xeb\xaf\xbb\xd7\xcd\xbd\xdf\x22\x68\x5b\xe5\xf3\xff\x4b\x1e" +
	"\xaa\x0f\xb5\x1c\xf6\xae\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\xfc\xeb\x0d\x9f\x8f\x99\x00\x00\x28\x00\x00"

var _ = Describe("Droplet", func() {
	var (
		droplet *Droplet
		logs    *gbytes.Buffer
	)

	BeforeEach(func() {
		client, err := docker.NewEnvClient()
		Expect(err).NotTo(HaveOccurred())
		logs = gbytes.NewBuffer()
		droplet = &Droplet{
			DiegoVersion: "0.1482.0",
			GoVersion:    "1.7",
			Docker:       client,
			Logs:         io.MultiWriter(logs, GinkgoWriter),
		}
	})

	Describe("#Build", func() {
		It("should build", func() {
			// TODO: spin up a test server and host a buildpack fixture
			droplet, err := droplet.Build("some-app", strings.NewReader(tgzFixture), []string{
				"https://github.com/cloudfoundry/staticfile-buildpack#v1.3.11",
			})
			Expect(err).NotTo(HaveOccurred())
			defer droplet.Close()
			data, _ := ioutil.ReadAll(droplet)
			fmt.Println(len(data))
			fmt.Println(string(logs.Contents()))
		})
	})
})
