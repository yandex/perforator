/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package platforms

import (
	"fmt"
	"sort"
	"testing"

	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// Test the platform compatibility of the different OS Versions
func Test_PlatformCompat(t *testing.T) {
	for _, tc := range []struct {
		hostOS    uint16
		ctrOS     uint16
		shouldRun bool
	}{
		{
			hostOS:    ltsc2019,
			ctrOS:     ltsc2019,
			shouldRun: true,
		},
		{
			hostOS:    ltsc2019,
			ctrOS:     ltsc2022,
			shouldRun: false,
		},
		{
			hostOS:    ltsc2022,
			ctrOS:     ltsc2019,
			shouldRun: false,
		},
		{
			hostOS:    ltsc2022,
			ctrOS:     ltsc2022,
			shouldRun: true,
		},
		{
			hostOS:    v22H2Win11,
			ctrOS:     ltsc2019,
			shouldRun: false,
		},
		{
			hostOS:    v22H2Win11,
			ctrOS:     ltsc2022,
			shouldRun: true,
		},
		{
			hostOS:    v23H2,
			ctrOS:     ltsc2019,
			shouldRun: false,
		},
		{
			hostOS:    v23H2,
			ctrOS:     ltsc2022,
			shouldRun: true,
		},
		{
			hostOS:    ltsc2025,
			ctrOS:     ltsc2022,
			shouldRun: true,
		},
		{
			hostOS:    ltsc2022,
			ctrOS:     ltsc2025,
			shouldRun: false,
		},
		{
			hostOS:    ltsc2022,
			ctrOS:     v22H2Win11,
			shouldRun: false,
		},
		{
			hostOS:    ltsc2025,
			ctrOS:     v22H2Win11,
			shouldRun: true,
		},
	} {
		t.Run(fmt.Sprintf("Host_%d_Ctr_%d", tc.hostOS, tc.ctrOS), func(t *testing.T) {
			hostOSVersion := windowsOSVersion{
				MajorVersion: 10,
				MinorVersion: 0,
				Build:        tc.hostOS,
			}
			ctrOSVersion := windowsOSVersion{
				MajorVersion: 10,
				MinorVersion: 0,
				Build:        tc.ctrOS,
			}
			if checkWindowsHostAndContainerCompat(hostOSVersion, ctrOSVersion) != tc.shouldRun {
				var expectedResultStr string
				if !tc.shouldRun {
					expectedResultStr = " NOT"
				}
				t.Fatalf("host %v should%s be able to run guest %v", tc.hostOS, expectedResultStr, tc.ctrOS)
			}
		})
	}
}

func Test_PlatformOrder(t *testing.T) {
	linuxPlatform := specs.Platform{
		Architecture: "amd64",
		OS:           "linux",
		OSVersion:    "",
		OSFeatures:   nil,
		Variant:      "",
	}
	ws2022Platform := specs.Platform{
		Architecture: "amd64",
		OS:           "windows",
		OSVersion:    "10.0.20348.3091",
		OSFeatures:   nil,
		Variant:      "",
	}
	ws2025Platform := specs.Platform{
		Architecture: "amd64",
		OS:           "windows",
		OSVersion:    "10.0.26100.2894",
		OSFeatures:   nil,
		Variant:      "",
	}
	ws2025Rev3000Platform := specs.Platform{
		Architecture: "amd64",
		OS:           "windows",
		OSVersion:    "10.0.26100.3000",
		OSFeatures:   nil,
		Variant:      "",
	}

	tt := []struct {
		name         string
		hostPlatform specs.Platform
		platforms    []specs.Platform
		wantPlatform specs.Platform
	}{
		{
			name:         "Windows Server 2022 should select 2022",
			hostPlatform: ws2022Platform,
			platforms:    []specs.Platform{linuxPlatform, ws2022Platform, ws2025Platform},
			wantPlatform: ws2022Platform,
		},
		{
			name:         "Windows Server 2025 should select 2025",
			hostPlatform: ws2025Platform,
			platforms:    []specs.Platform{linuxPlatform, ws2022Platform, ws2025Platform},
			wantPlatform: ws2025Platform,
		},
		{
			name:         "Windows Server 2025 should select 2025 latest rev",
			hostPlatform: ws2025Platform,
			platforms:    []specs.Platform{linuxPlatform, ws2022Platform, ws2025Rev3000Platform},
			wantPlatform: ws2025Rev3000Platform,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			comparer := &windowsMatchComparer{Matcher: NewMatcher(tc.hostPlatform)}

			sort.SliceStable(tc.platforms, func(i, j int) bool {
				return comparer.Less(tc.platforms[i], tc.platforms[j])
			})

			if tc.platforms[0].OS != tc.wantPlatform.OS || tc.platforms[0].OSVersion != tc.wantPlatform.OSVersion {
				t.Errorf("Platform mismatch, want %q/%q, got %q/%q", tc.wantPlatform.OS, tc.wantPlatform.OSVersion, tc.platforms[0].OS, tc.platforms[0].OSVersion)
			}
		})
	}

}
