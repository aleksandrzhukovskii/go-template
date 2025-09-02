package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/aleksandrzhukovskii/go-template/internal/config"
)

type TestSuite struct {
	suite.Suite
	variables map[string]string
}

func (s *TestSuite) SetupTest() {
	s.variables = map[string]string{
		"IP":     "127.0.0.1",
		"PORT":   "8080",
		"DB":     "db",
		"SERVER": "server",
	}
	for k, v := range s.variables {
		if err := os.Setenv(k, v); err != nil {
			s.Fail(err.Error())
		}
	}
}

func (s *TestSuite) TestParse_MissedValue() {
	if err := os.Unsetenv("DB"); err != nil {
		s.Fail(err.Error())
	}
	cfg, err := config.New()
	s.Equal(config.Config{}, cfg)
	s.Error(err, "")
}

func (s *TestSuite) TestParse_OK() {
	cfg, err := config.New()
	s.NoError(err)
	s.Equal(s.variables["IP"], cfg.HttpGrpc.IP)
	s.Equal(s.variables["PORT"], cfg.HttpGrpc.Port)
	s.NotEmpty(cfg.SqLite.Path)
	s.True(strings.HasSuffix(cfg.SqLite.Path, "/db"))
	s.DirExists(strings.TrimSuffix(cfg.SqLite.Path, "/db"))
}

func TestParse(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
