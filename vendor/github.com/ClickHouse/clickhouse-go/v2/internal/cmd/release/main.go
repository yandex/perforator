// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var skipWorkingTreeIsDirtyCheck = flag.Bool("skip-working-tree-is-dirty-check", false, "Skip working tree is dirty check")

func main() {
	flag.Parse()

	if !(*skipWorkingTreeIsDirtyCheck) && gitRepositoryWorkingTreeIsDirty() {
		log.Fatalln("Git working tree is dirty")
		return
	}

	releaseURL := getLatestDraftReleaseURL()

	log.Println("Latest draft release URL:")
	log.Println(releaseURL)

	r := getRelease(releaseURL)

	log.Println("Release tag:")
	log.Println(r.TagName)
	log.Println("Release body:")
	log.Println(r.Body)

	major, minor, patch, err := parseSemVer(r.TagName)
	if err != nil {
		log.Fatalln(err)
		return
	}

	if len(r.Body) == 0 {
		log.Fatalln("Release body is empty")
		return
	}

	changelogPath := changelogFilePath()
	prependReleaseToChangelog(changelogPath, r)

	if err := updateClientInfo(major, minor, patch); err != nil {
		log.Fatalln(err)
		return
	}

	runGoGenerate()
	runGoFmt()

	gitHubOutputReleaseURLIfAvailable(releaseURL)
}

func gitHubOutputReleaseURLIfAvailable(url string) {
	if len(url) == 0 {
		return
	}

	gitHubOutputFile, exists := os.LookupEnv("GITHUB_OUTPUT")
	if !exists {
		return
	}

	f, err := os.OpenFile(gitHubOutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalln(err)
		return
	}

	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	if _, err := f.WriteString(fmt.Sprintf("RELEASE_URL=%s\n", url)); err != nil {
		log.Fatalln(err)
		return
	}
}

func runGoFmt() {
	cmd := exec.Command("go", "fmt", "./...")
	cmd.Dir = getRootPath()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
		return
	}
}

func runGoGenerate() {
	cmd := exec.Command("go", "generate", "./...")
	cmd.Dir = getRootPath()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
		return
	}
}

func gitRepositoryWorkingTreeIsDirty() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		log.Fatalln(err)
		return false
	}
	return len(out) > 0
}

func parseSemVer(version string) (major, minor, patch int, err error) {
	// Define a regular expression to match SemVer format
	re := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

	// Apply the regular expression to the version string
	match := re.FindStringSubmatch(version)

	// Check if the version string matches the SemVer format
	if len(match) != 4 {
		err = fmt.Errorf("invalid SemVer format: %s", version)
		return
	}

	// Parse the major, minor, and patch components as integers
	major, err = strconv.Atoi(match[1])
	if err != nil {
		return
	}

	minor, err = strconv.Atoi(match[2])
	if err != nil {
		return
	}

	patch, err = strconv.Atoi(match[3])
	if err != nil {
		return
	}

	return
}

func prependReleaseToChangelog(changelogPath string, r release) {
	f, err := os.OpenFile(changelogPath, os.O_RDWR, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		log.Fatalln(err)
		return
	}

	f.Seek(0, io.SeekStart)
	f.WriteString(fmt.Sprintf("# %s, %s ", r.TagName, time.Now().Format("2006-01-02")))
	f.WriteString(r.Body)
	f.WriteString("\n\n")
	f.Write(content)
}

func changelogFilePath() string {
	rootPath := getRootPath()
	changelogPath := rootPath + "/CHANGELOG.md"
	return changelogPath
}

func getRootPath() string {
	wd, _ := os.Getwd()
	rootPath := strings.Replace(wd, "internal/cmd/release", "", 1)
	return rootPath
}

func getRelease(releaseURL string) release {
	req, err := http.NewRequest(http.MethodGet, releaseURL, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("Authorization", "token "+os.Getenv("GITHUB_TOKEN"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	var release release
	if err := json.NewDecoder(res.Body).Decode(&release); err != nil {
		log.Fatalln(err)
	}

	return release
}

type release struct {
	URL     string `json:"url"`
	Body    string `json:"body"`
	TagName string `json:"tag_name"`
}

func getLatestDraftReleaseURL() string {
	// Fetch the latest release from GitHub repository using GitHub API
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/clickhouse/clickhouse-go/releases?per_page=100", nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("Authorization", "token "+os.Getenv("GITHUB_TOKEN"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer res.Body.Close()

	var releases []struct {
		URL   string `json:"url"`
		Draft bool   `json:"draft"`
	}
	if err := json.NewDecoder(res.Body).Decode(&releases); err != nil {
		log.Fatalln(err)
	}

	// filter out releases that are not drafts
	for i := 0; i < len(releases); {
		if releases[i].Draft {
			return releases[i].URL
		}
	}

	log.Fatalln("No draft releases found")
	return ""
}

func updateClientInfo(major, minor, patch int) error {
	// Open the client_info.go file for reading and writing
	file, err := os.OpenFile(getRootPath()+"/client_info.go", os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read the contents of the file into memory
	bytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// Replace the ClientVersionMajor, ClientVersionMinor, and ClientVersionPatch lines
	reMajor := regexp.MustCompile(`ClientVersionMajor\s+=\s+\d+`)
	reMinor := regexp.MustCompile(`ClientVersionMinor\s+=\s+\d+`)
	rePatch := regexp.MustCompile(`ClientVersionPatch\s+=\s+\d+`)
	newLines := []string{
		fmt.Sprintf("ClientVersionMajor       = %d", major),
		fmt.Sprintf("ClientVersionMinor       = %d", minor),
		fmt.Sprintf("ClientVersionPatch       = %d", patch),
	}
	scanner := bufio.NewScanner(strings.NewReader(string(bytes)))
	var newContent string
	for scanner.Scan() {
		line := scanner.Text()
		if reMajor.MatchString(line) {
			line = newLines[0]
		} else if reMinor.MatchString(line) {
			line = newLines[1]
		} else if rePatch.MatchString(line) {
			line = newLines[2]
		}
		newContent += line + "\n"
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Write the updated content back to the file
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	if err := file.Truncate(0); err != nil {
		return err
	}
	if _, err := file.Write([]byte(newContent)); err != nil {
		return err
	}

	return nil
}
