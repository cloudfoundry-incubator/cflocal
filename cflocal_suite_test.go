package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCFLocal(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CF Local Suite")
}
