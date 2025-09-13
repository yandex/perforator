package yatool_test

import (
	"io"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	arcadiaRoot string
}

func (s *TestSuite) SetupSuite() {
	newArcadiaRoot, err := os.MkdirTemp("", "yatool-mini-arcadia-*")
	s.Require().NoError(err, "create mini arcadia root")

	miniArc, err := filepath.Abs("./testdata/mini_arcadia")
	s.Require().NoError(err, "resolve testdata mini arcadia")

	err = copyTree(miniArc, newArcadiaRoot)
	s.Require().NoError(err, "copy mini arcadia")

	s.arcadiaRoot = newArcadiaRoot
}

func (s *TestSuite) TearDownSuite() {
	_ = os.RemoveAll(s.arcadiaRoot)
}

func (s *TestSuite) InRoot(subpath ...string) string {
	return filepath.Join(append([]string{s.arcadiaRoot}, subpath...)...)
}

func copyFile(from, to string) error {
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(from)
		if err != nil {
			return err
		}
		return os.Symlink(linkTarget, to)
	}

	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(to)
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()

	_, err = io.Copy(dst, src)
	return err
}

func copyTree(from, to string) error {
	return filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(to, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		return copyFile(path, destPath)
	})
}
