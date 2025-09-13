package yatool_test

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/yandex/perforator/library/go/yatool"
)

func ExampleFindArcadiaRoot() {
	arcadiaRoot, err := yatool.FindArcadiaRoot("arcadia/path")
	if err != nil {
		log.Fatalf("failed to find Arcadia root: %s\n", err)
	}

	cwd, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("failed to resolve current working dir: %s\n", err)
	}

	projectPath, err := filepath.Rel(arcadiaRoot, cwd)
	if err != nil {
		log.Fatalf("failed to resolve project path: %s\n", err)
	}

	fmt.Printf("Arcadia root: %s\n", arcadiaRoot)
	fmt.Printf("Current working dir: %s\n", cwd)
	fmt.Printf("Project path: %s\n", projectPath)
}

func ExampleArcadiaRoot() {
	arcadiaRoot, err := yatool.ArcadiaRoot()
	if err != nil {
		log.Fatalf("failed to find Arcadia root: %s\n", err)
	}

	cwd, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("failed to resolve current working dir: %s\n", err)
	}

	projectPath, err := filepath.Rel(arcadiaRoot, cwd)
	if err != nil {
		log.Fatalf("failed to resolve project path: %s\n", err)
	}

	fmt.Printf("Arcadia root: %s\n", arcadiaRoot)
	fmt.Printf("Current working dir: %s\n", cwd)
	fmt.Printf("Project path: %s\n", projectPath)
}
