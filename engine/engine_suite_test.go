package engine_test

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEngine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Engine Suite")
}

var client *docker.Client

func setupClient() {
	if client != nil {
		return
	}
	var err error
	client, err = docker.NewEnvClient()
	Expect(err).NotTo(HaveOccurred())
	client.UpdateClientVersion("")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	setupClient()
	ctx := context.Background()
	body, err := client.ImagePull(ctx, "sclevine/test", types.ImagePullOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(ioutil.ReadAll(body)).NotTo(BeZero())
	return nil
}, func(_ []byte) {
	setupClient()
})

var _ = SynchronizedAfterSuite(func() {
	Expect(client.Close()).To(Succeed())
}, func() {})
