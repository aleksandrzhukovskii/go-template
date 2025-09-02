package e2e_tests

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestRun(t *testing.T) {
	if os.Getenv("TYPE") == "http" {
		suite.Run(t, &HTTPSuite{})
	} else if os.Getenv("TYPE") == "graphql" {
		suite.Run(t, &GraphSuite{})
	} else if os.Getenv("TYPE") == "grpc" {
		suite.Run(t, &GrpcSuite{})
	}
}
