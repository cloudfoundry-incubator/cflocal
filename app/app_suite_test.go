package app_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App Suite")
}

func percentColor(format string, a ...interface{}) string {
	return fmt.Sprintf(format+"%% ", a...)
}
