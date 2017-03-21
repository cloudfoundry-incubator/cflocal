package local_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLocal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Local Suite")
}

func percentColor(format string, a ...interface{}) string {
	return fmt.Sprintf(format+"%% ", a...)
}
