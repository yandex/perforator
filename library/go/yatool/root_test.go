package yatool_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/yandex/perforator/library/go/yatool"
)

func TestFindRoot(t *testing.T) {
	suite.Run(t, new(FindRootSuite))
}

type FindRootSuite struct {
	TestSuite
}

func (s *FindRootSuite) TestInRoot() {
	actualRoot, err := yatool.FindArcadiaRoot(s.arcadiaRoot)
	s.Require().NoError(err, "find arcadia root")
	s.Require().Equal(s.arcadiaRoot, actualRoot)
}

func (s *FindRootSuite) TestInNestedProject() {
	actualRoot, err := yatool.FindArcadiaRoot(s.InRoot("test", "nested"))
	s.Require().NoError(err, "find arcadia root")
	s.Require().Equal(s.arcadiaRoot, actualRoot)
}

func (s *FindRootSuite) TestBySymlink() {
	linkRoot, err := os.MkdirTemp("", "arc-link")
	s.Require().NoError(err, "create temp dir for link")

	defer func() {
		_ = os.RemoveAll(linkRoot)
	}()

	linkPath := filepath.Join(linkRoot, "project")
	err = os.Symlink(s.InRoot("test", "nested"), linkPath)
	s.Require().NoError(err, "create link from temp dir to arcadia")

	actualRoot, err := yatool.FindArcadiaRoot(linkPath)
	s.Require().NoError(err, "find arcadia root")
	s.Require().Equal(s.arcadiaRoot, actualRoot)
}

func (s *FindRootSuite) TestFail() {
	fakeArcadia, err := os.MkdirTemp("", "arc-root")
	s.Require().NoError(err, "create temp arcadia root")

	defer func() {
		_ = os.RemoveAll(fakeArcadia)
	}()

	_, err = yatool.FindArcadiaRoot(fakeArcadia)
	s.Require().Error(err, "found unexpected arcadia root")
}
