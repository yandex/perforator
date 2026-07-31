/*
Copyright 2018 The Kubernetes Authors.

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

package net

import (
	"fmt"
	"net"
	"testing"
)

func TestDualStackIPs(t *testing.T) {
	testCases := []struct {
		ips            []string
		errMessage     string
		expectedResult bool
		expectError    bool
	}{
		{
			ips:            []string{"1.1.1.1"},
			errMessage:     "should fail because length is not at least 2",
			expectedResult: false,
			expectError:    false,
		},
		{
			ips:            []string{},
			errMessage:     "should fail because length is not at least 2",
			expectedResult: false,
			expectError:    false,
		},
		{
			ips:            []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"},
			errMessage:     "should fail because all are v4",
			expectedResult: false,
			expectError:    false,
		},
		{
			ips:            []string{"fd92:20ba:ca:34f7:ffff:ffff:ffff:ffff", "fd92:20ba:ca:34f7:ffff:ffff:ffff:fff0", "fd92:20ba:ca:34f7:ffff:ffff:ffff:fff1"},
			errMessage:     "should fail because all are v6",
			expectedResult: false,
			expectError:    false,
		},
		{
			ips:            []string{"1.1.1.1", "not-a-valid-ip"},
			errMessage:     "should fail because 2nd ip is invalid",
			expectedResult: false,
			expectError:    true,
		},
		{
			ips:            []string{"not-a-valid-ip", "fd92:20ba:ca:34f7:ffff:ffff:ffff:ffff"},
			errMessage:     "should fail because 1st ip is invalid",
			expectedResult: false,
			expectError:    true,
		},
		{
			ips:            []string{"1.1.1.1", "fd92:20ba:ca:34f7:ffff:ffff:ffff:ffff"},
			errMessage:     "expected success, but found failure",
			expectedResult: true,
			expectError:    false,
		},
		{
			ips:            []string{"fd92:20ba:ca:34f7:ffff:ffff:ffff:ffff", "1.1.1.1", "fd92:20ba:ca:34f7:ffff:ffff:ffff:fff0"},
			errMessage:     "expected success, but found failure",
			expectedResult: true,
			expectError:    false,
		},
		{
			ips:            []string{"1.1.1.1", "fd92:20ba:ca:34f7:ffff:ffff:ffff:ffff", "10.0.0.0"},
			errMessage:     "expected success, but found failure",
			expectedResult: true,
			expectError:    false,
		},
		{
			ips:            []string{"fd92:20ba:ca:34f7:ffff:ffff:ffff:ffff", "1.1.1.1"},
			errMessage:     "expected success, but found failure",
			expectedResult: true,
			expectError:    false,
		},
	}
	// for each test case, test the regular func and the string func
	for _, tc := range testCases {
		dualStack, err := IsDualStackIPStrings(tc.ips)
		if err == nil && tc.expectError {
			t.Errorf("%s", tc.errMessage)
			continue
		}
		if err != nil && !tc.expectError {
			t.Errorf("failed to run test case for %v, error: %v", tc.ips, err)
			continue
		}
		if dualStack != tc.expectedResult {
			t.Errorf("%v for %v", tc.errMessage, tc.ips)
		}
	}

	for _, tc := range testCases {
		ips := make([]net.IP, 0, len(tc.ips))
		for _, ip := range tc.ips {
			parsedIP := ParseIPSloppy(ip)
			ips = append(ips, parsedIP)
		}
		dualStack, err := IsDualStackIPs(ips)
		if err == nil && tc.expectError {
			t.Errorf("%s", tc.errMessage)
			continue
		}
		if err != nil && !tc.expectError {
			t.Errorf("failed to run test case for %v, error: %v", tc.ips, err)
			continue
		}
		if dualStack != tc.expectedResult {
			t.Errorf("%v for %v", tc.errMessage, tc.ips)
		}
	}
}

func TestDualStackCIDRs(t *testing.T) {
	testCases := []struct {
		cidrs          []string
		errMessage     string
		expectedResult bool
		expectError    bool
	}{
		{
			cidrs:          []string{"10.10.10.10/8"},
			errMessage:     "should fail because length is not at least 2",
			expectedResult: false,
			expectError:    false,
		},
		{
			cidrs:          []string{},
			errMessage:     "should fail because length is not at least 2",
			expectedResult: false,
			expectError:    false,
		},
		{
			cidrs:          []string{"10.10.10.10/8", "20.20.20.20/8", "30.30.30.30/8"},
			errMessage:     "should fail because all cidrs are v4",
			expectedResult: false,
			expectError:    false,
		},
		{
			cidrs:          []string{"2000::/10", "3000::/10"},
			errMessage:     "should fail because all cidrs are v6",
			expectedResult: false,
			expectError:    false,
		},
		{
			cidrs:          []string{"10.10.10.10/8", "not-a-valid-cidr"},
			errMessage:     "should fail because 2nd cidr is invalid",
			expectedResult: false,
			expectError:    true,
		},
		{
			cidrs:          []string{"not-a-valid-ip", "2000::/10"},
			errMessage:     "should fail because 1st cidr is invalid",
			expectedResult: false,
			expectError:    true,
		},
		{
			cidrs:          []string{"10.10.10.10/8", "2000::/10"},
			errMessage:     "expected success, but found failure",
			expectedResult: true,
			expectError:    false,
		},
		{
			cidrs:          []string{"2000::/10", "10.10.10.10/8"},
			errMessage:     "expected success, but found failure",
			expectedResult: true,
			expectError:    false,
		},
		{
			cidrs:          []string{"2000::/10", "10.10.10.10/8", "3000::/10"},
			errMessage:     "expected success, but found failure",
			expectedResult: true,
			expectError:    false,
		},
	}

	// for each test case, test the regular func and the string func
	for _, tc := range testCases {
		dualStack, err := IsDualStackCIDRStrings(tc.cidrs)
		if err == nil && tc.expectError {
			t.Errorf("%s", tc.errMessage)
			continue
		}
		if err != nil && !tc.expectError {
			t.Errorf("failed to run test case for %v, error: %v", tc.cidrs, err)
			continue
		}
		if dualStack != tc.expectedResult {
			t.Errorf("%v for %v", tc.errMessage, tc.cidrs)
		}
	}

	for _, tc := range testCases {
		cidrs := make([]*net.IPNet, 0, len(tc.cidrs))
		for _, cidr := range tc.cidrs {
			_, parsedCIDR, _ := ParseCIDRSloppy(cidr)
			cidrs = append(cidrs, parsedCIDR)
		}

		dualStack, err := IsDualStackCIDRs(cidrs)
		if err == nil && tc.expectError {
			t.Errorf("%s", tc.errMessage)
			continue
		}
		if err != nil && !tc.expectError {
			t.Errorf("failed to run test case for %v, error: %v", tc.cidrs, err)
			continue
		}
		if dualStack != tc.expectedResult {
			t.Errorf("%v for %v", tc.errMessage, tc.cidrs)
		}
	}
}

func checkOneIPFamily(t *testing.T, ip string, expectedFamily, family IPFamily, isIPv4, isIPv6 bool) {
	t.Helper()
	if family != expectedFamily {
		t.Errorf("Expect %q family %q, got %q", ip, expectedFamily, family)
	}
	if isIPv4 != (expectedFamily == IPv4) {
		t.Errorf("Expect %q ipv4 %v, got %v", ip, expectedFamily == IPv4, isIPv6)
	}
	if isIPv6 != (expectedFamily == IPv6) {
		t.Errorf("Expect %q ipv6 %v, got %v", ip, expectedFamily == IPv6, isIPv6)
	}
}

func TestIPFamilyOf(t *testing.T) {
	// See test cases in ips_test.go
	for _, tc := range goodTestIPs {
		if tc.skipFamily {
			continue
		}
		t.Run(tc.desc, func(t *testing.T) {
			for _, str := range tc.strings {
				family := IPFamilyOfString(str)
				isIPv4 := IsIPv4String(str)
				isIPv6 := IsIPv6String(str)
				checkOneIPFamily(t, str, tc.family, family, isIPv4, isIPv6)
			}
			for _, ip := range tc.ips {
				family := IPFamilyOf(ip)
				isIPv4 := IsIPv4(ip)
				isIPv6 := IsIPv6(ip)
				checkOneIPFamily(t, ip.String(), tc.family, family, isIPv4, isIPv6)
			}
		})
	}

	// See test cases in ips_test.go
	for _, tc := range badTestIPs {
		if tc.skipFamily {
			continue
		}
		t.Run(tc.desc, func(t *testing.T) {
			for _, ip := range tc.ips {
				family := IPFamilyOf(ip)
				isIPv4 := IsIPv4(ip)
				isIPv6 := IsIPv6(ip)
				checkOneIPFamily(t, fmt.Sprintf("%#v", ip), IPFamilyUnknown, family, isIPv4, isIPv6)
			}
			for _, str := range tc.strings {
				family := IPFamilyOfString(str)
				isIPv4 := IsIPv4String(str)
				isIPv6 := IsIPv6String(str)
				checkOneIPFamily(t, str, IPFamilyUnknown, family, isIPv4, isIPv6)
			}
		})
	}
}

func TestIPFamilyOfCIDR(t *testing.T) {
	// See test cases in ips_test.go
	for _, tc := range goodTestCIDRs {
		if tc.skipFamily {
			continue
		}
		t.Run(tc.desc, func(t *testing.T) {
			for _, str := range tc.strings {
				family := IPFamilyOfCIDRString(str)
				isIPv4 := IsIPv4CIDRString(str)
				isIPv6 := IsIPv6CIDRString(str)
				checkOneIPFamily(t, str, tc.family, family, isIPv4, isIPv6)
			}
			for _, ipnet := range tc.ipnets {
				family := IPFamilyOfCIDR(ipnet)
				isIPv4 := IsIPv4CIDR(ipnet)
				isIPv6 := IsIPv6CIDR(ipnet)
				checkOneIPFamily(t, ipnet.String(), tc.family, family, isIPv4, isIPv6)
			}
		})
	}

	// See test cases in ips_test.go
	for _, tc := range badTestCIDRs {
		if tc.skipFamily {
			continue
		}
		t.Run(tc.desc, func(t *testing.T) {
			for _, ipnet := range tc.ipnets {
				family := IPFamilyOfCIDR(ipnet)
				isIPv4 := IsIPv4CIDR(ipnet)
				isIPv6 := IsIPv6CIDR(ipnet)
				str := "<nil>"
				if ipnet != nil {
					str = fmt.Sprintf("%#v", *ipnet)
				}
				checkOneIPFamily(t, str, IPFamilyUnknown, family, isIPv4, isIPv6)
			}
			for _, str := range tc.strings {
				family := IPFamilyOfCIDRString(str)
				isIPv4 := IsIPv4CIDRString(str)
				isIPv6 := IsIPv6CIDRString(str)
				checkOneIPFamily(t, str, IPFamilyUnknown, family, isIPv4, isIPv6)
			}
		})
	}
}
