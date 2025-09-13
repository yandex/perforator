package yatool_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/yandex/perforator/library/go/yatool"
)

func TestYaPath(t *testing.T) {
	suite.Run(t, new(YaPathSuite))
}

type YaPathSuite struct {
	TestSuite
	yaPath string
}

func (s *YaPathSuite) SetupSuite() {
	s.TestSuite.SetupSuite()
	s.yaPath = s.InRoot("ya")
}

func (s *YaPathSuite) TestInRoot() {
	actualYa, err := yatool.FindYa(s.arcadiaRoot)
	s.Require().NoError(err, "find ya")
	s.Require().Equal(s.yaPath, actualYa)
}

func (s *YaPathSuite) TestInNestedProject() {
	actualYa, err := yatool.FindYa(s.InRoot("test", "nested"))
	s.Require().NoError(err, "find ya")
	s.Require().Equal(s.yaPath, actualYa)
}

func (s *YaPathSuite) TestRelative() {
	wd, err := os.Getwd()
	s.Require().NoError(err, "get cwd")
	defer func() {
		_ = os.Chdir(wd)
	}()

	err = os.Chdir(s.InRoot("test", "nested"))
	s.Require().NoError(err, "chdir into tested dir")

	actualYa, err := yatool.FindYa(".")
	s.Require().NoError(err, "find ya")
	s.Require().Equal(s.yaPath, actualYa)
}

func (s *YaPathSuite) TestFail() {
	fakeArcadia, err := os.MkdirTemp("", "arc-root")
	s.Require().NoError(err, "create empty arcadia root")

	defer func() {
		_ = os.RemoveAll(fakeArcadia)
	}()

	_, err = yatool.FindYa(fakeArcadia)
	s.Require().Error(err, "ya must not exist in the fake Arcadia root")
}
