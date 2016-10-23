package local_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLocal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Local Suite")
}

func percentColor(format string, a ...interface{}) string {
	return fmt.Sprintf(format+"%% ", a...)
}
