/*
Copyright 2025 The Kubernetes Authors.

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
	"net"
	"testing"
)

// testIP represents a set of equivalent IP address representations.
type testIP struct {
	desc    string
	family  IPFamily
	strings []string
	ips     []net.IP

	skipFamily bool
	skipParse  bool
}

// goodTestIPs are "good" test IP values. For each item:
//
// Preconditions (not involving functions in netutils):
//   - Each element of .ips is the same (i.e., .Equal()).
//   - Each element of .ips stringifies to .strings[0].
//
// IPFamily tests (unless `skipFamily: true`):
//   - Each element of .strings should be identified as .family.
//   - Each element of .ips should be identified as .family.
//
// Parsing tests (unless `skipParse: true`):
//   - Each element of .strings should parse to a value equal to .ips[0].
var goodTestIPs = []testIP{
	{
		desc:   "IPv4",
		family: IPv4,
		strings: []string{
			"192.168.0.5",
			"192.168.000.005",
		},
		ips: []net.IP{
			net.IPv4(192, 168, 0, 5),
			{192, 168, 0, 5},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xFF, 0xFF, 192, 168, 0, 5},
			net.ParseIP("192.168.0.5"),
			func() net.IP { ip, _, _ := net.ParseCIDR("192.168.0.5/24"); return ip }(),
			func() net.IP { _, ipnet, _ := net.ParseCIDR("192.168.0.5/32"); return ipnet.IP }(),
		},
	},
	{
		desc:   "IPv4 all-zeros",
		family: IPv4,
		strings: []string{
			"0.0.0.0",
			"000.000.000.000",
		},
		ips: []net.IP{
			net.IPv4zero,
			net.IPv4(0, 0, 0, 0),
			{0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xFF, 0xFF, 0, 0, 0, 0},
			net.ParseIP("0.0.0.0"),
		},
	},
	{
		desc:   "IPv4 broadcast",
		family: IPv4,
		strings: []string{
			"255.255.255.255",
		},
		ips: []net.IP{
			net.IPv4bcast,
			net.IPv4(255, 255, 255, 255),
			{255, 255, 255, 255},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xFF, 0xFF, 255, 255, 255, 255},
			net.ParseIP("255.255.255.255"),
			// A /32 IPMask is equivalent to 255.255.255.255
			func() net.IP { _, ipnet, _ := net.ParseCIDR("1.2.3.4/32"); return net.IP(ipnet.Mask) }(),
		},
	},
	{
		desc:   "IPv6",
		family: IPv6,
		strings: []string{
			"2001:db8::5",
			"2001:0db8::0005",
			"2001:DB8::5",
		},
		ips: []net.IP{
			{0x20, 0x01, 0x0D, 0xB8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x05},
			net.ParseIP("2001:db8::5"),
			func() net.IP { ip, _, _ := net.ParseCIDR("2001:db8::5/64"); return ip }(),
			func() net.IP { _, ipnet, _ := net.ParseCIDR("2001:db8::5/128"); return ipnet.IP }(),
		},
	},
	{
		desc:   "IPv6 all-zeros",
		family: IPv6,
		strings: []string{
			"::",
			"0:0:0:0:0:0:0:0",
			"0000:0000:0000:0000:0000:0000:0000:0000",
			"0::0",
		},
		ips: []net.IP{
			net.IPv6zero,
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			net.ParseIP("::"),
			// ::/0 has an IP, network base IP, and Mask that are all
			// equivalent to ::
			func() net.IP { ip, _, _ := net.ParseCIDR("::/0"); return ip }(),
			func() net.IP { _, ipnet, _ := net.ParseCIDR("::/0"); return ipnet.IP }(),
			func() net.IP { _, ipnet, _ := net.ParseCIDR("::/0"); return net.IP(ipnet.Mask) }(),
		},
	},
	{
		desc:   "IPv6 loopback",
		family: IPv6,
		strings: []string{
			"::1",
			"0000:0000:0000:0000:0000:0000:0000:0001",
		},
		ips: []net.IP{
			net.IPv6loopback,
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			net.ParseIP("::1"),
		},
	},
	{
		desc: "IPv4-mapped IPv6",
		// net.IP can represent an IPv4 address internally as either a 4-byte
		// value or a 16-byte value, but it treats the two forms as equivalent.
		//
		// This test case confirms that:
		//   - The 4-byte and 16-byte forms of a given net.IP compare as .Equal().
		//   - Our parsers parse the plain IPv4 and IPv4-mapped IPv6 forms of an
		//     IPv4 string to the same thing.
		//   - The 4-byte and 16-byte forms of a given net.IP all stringify
		//     to the plain IPv4 string form (i.e., .strings[0]).
		family: IPv4,
		strings: []string{
			"192.168.0.5",
			"::ffff:192.168.0.5",
			"::ffff:0192.0168.0000.0005",
		},
		ips: []net.IP{
			{192, 168, 0, 5},
			{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xFF, 0xFF, 192, 168, 0, 5},
			net.IPv4(192, 168, 0, 5).To4(),
			net.IPv4(192, 168, 0, 5).To16(),
			net.ParseIP("192.168.0.5").To4(),
			net.ParseIP("192.168.0.5").To16(),
			net.ParseIP("::ffff:192.168.0.5").To4(),
			net.ParseIP("::ffff:192.168.0.5").To16(),
		},
	},
}

// badTestIPs are bad test IP values. For each item:
//
// IPFamily tests (unless `skipFamily: true`):
//   - Each element of .strings should be identified as IPFamilyUnknown.
//   - Each element of .ips should be identified as IPFamilyUnknown.
//
// Parsing tests (unless `skipParse: true`):
//   - Each element of .strings should fail to parse.
//   - Each element of .ips should stringify to an error value that fails to parse.
var badTestIPs = []testIP{
	{
		desc: "empty string is not an IP",
		strings: []string{
			"",
		},
	},
	{
		desc: "random non-IP string is not an IP",
		strings: []string{
			"bad ip",
		},
	},
	{
		desc: "domain name is not an IP",
		strings: []string{
			"www.example.com",
		},
	},
	{
		desc: "mangled IPv4 addresses are invalid",
		strings: []string{
			"1.2.3.400",
			"1.2..4",
			"1.2.3",
			"1.2.3.4.5",
		},
	},
	{
		desc: "mangled IPv6 addresses are invalid",
		strings: []string{
			"1:2::12345",
			"1::2::3",
			"1:2:::3",
			"1:2:3",
			"1:2:3:4:5:6:7:8:9",
			"1:2:3:4::6:7:8:9",
		},
	},
	{
		desc: "IPs do not have ports or brackets",
		strings: []string{
			"1.2.3.4:80",
			"[2001:db8::5]",
			"[2001:db8::5]:80",
			"www.example.com:80",
		},
	},
	{
		desc: "IPs with zones are invalid",
		strings: []string{
			"169.254.169.254%eth0",
			"fe80::1234%eth0",
		},
	},
	{
		desc: "CIDR strings are not IPs",
		strings: []string{
			"1.2.3.0/24",
			"2001:db8::/64",
		},
	},
	{
		desc: "IPs with whitespace are invalid",
		strings: []string{
			" 1.2.3.4",
			"1.2.3.4 ",
			" 2001:db8::5",
			"2001:db8::5 ",
		},
	},
	{
		desc: "nil is an invalid net.IP",
		ips: []net.IP{
			nil,
		},
	},
	{
		desc: "a byte slice of length other than 4 or 16 is an invalid net.IP",
		ips: []net.IP{
			{},
			{1, 2, 3},
			{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18},
		},
	},
}

// testCIDR represents a set of equivalent CIDR representations.
type testCIDR struct {
	desc    string
	family  IPFamily
	strings []string
	ipnets  []*net.IPNet

	skipFamily bool
	skipParse  bool
}

// goodTestCIDRs are "good" test CIDR values. For each item:
//
// Preconditions:
//   - Each element of .ipnets stringifies to .strings[0].
//
// IPFamily tests (unless `skipFamily: true`):
//   - Each element of .strings should be identified as .family.
//   - Each element of .ipnets should be identified as .family.
//
// Parsing tests (unless `skipParse: true`):
//   - Each element of .strings should parse to a value "equal" to .ipnets[0].
//
// (Unlike net.IP, *net.IPNet has no `.Equal()` method, and testing equality "by hand" is
// complicated (there are 4 equivalent representations of every IPv4 CIDR value), so we
// just consider two *net.IPNet values to be equal if they stringify to the same value.)
var goodTestCIDRs = []testCIDR{
	{
		desc:   "IPv4",
		family: IPv4,
		strings: []string{
			"1.2.3.0/24",
		},
		ipnets: []*net.IPNet{
			{IP: net.IPv4(1, 2, 3, 0), Mask: net.CIDRMask(24, 32)},
			{IP: net.ParseIP("1.2.3.0"), Mask: net.CIDRMask(24, 32)},
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("1.2.3.0/24"); return ipnet }(),
		},
	},
	{
		desc:   "IPv4, single IP",
		family: IPv4,
		strings: []string{
			"1.1.1.1/32",
		},
		ipnets: []*net.IPNet{
			{IP: net.IPv4(1, 1, 1, 1), Mask: net.CIDRMask(32, 32)},
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("1.1.1.1/32"); return ipnet }(),
		},
	},
	{
		desc:   "IPv4, all IPs",
		family: IPv4,
		strings: []string{
			"0.0.0.0/0",
			"000.000.000.000/000",
		},
		ipnets: []*net.IPNet{
			{IP: net.IPv4zero.To4(), Mask: net.IPMask(net.IPv4zero.To4())},
			{IP: net.IPv4(0, 0, 0, 0), Mask: net.CIDRMask(0, 32)},
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("0.0.0.0/0"); return ipnet }(),
		},
	},
	{
		desc: "IPv4 ifaddr (masked)",
		// This tests that if you try to parse an "ifaddr-style" CIDR string with
		// ParseCIDR/ParseCIDRSloppy, the *net.IPNet return value has the bits
		// beyond the prefix length masked out.
		family: IPv4,
		strings: []string{
			"1.2.3.0/24",
			"1.2.3.4/24",
			"1.2.3.255/24",
		},
		ipnets: []*net.IPNet{
			{IP: net.IPv4(1, 2, 3, 0), Mask: net.CIDRMask(24, 32)},
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("1.2.3.0/24"); return ipnet }(),
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("1.2.3.4/24"); return ipnet }(),
		},
	},
	{
		desc:   "IPv4 ifaddr",
		family: IPv4,
		strings: []string{
			"1.2.3.4/24",
		},
		ipnets: []*net.IPNet{
			{IP: net.IPv4(1, 2, 3, 4), Mask: net.CIDRMask(24, 32)},
		},

		// The *net.IPNet return value of ParseCIDRSloppy() masks out the lower
		// bits, so the parsed version won't compare equal to .ipnets[0]
		skipParse: true,
	},
	{
		desc:   "IPv6",
		family: IPv6,
		strings: []string{
			"2001:db8::/64",
			"2001:db8:0:0:0:0:0:0/64",
			"2001:DB8::/64",
		},
		ipnets: []*net.IPNet{
			{IP: net.IP{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, Mask: net.IPMask{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
			{IP: net.ParseIP("2001:db8::"), Mask: net.CIDRMask(64, 128)},
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("2001:db8::/64"); return ipnet }(),
		},
	},
	{
		desc:   "IPv6, all IPs",
		family: IPv6,
		strings: []string{
			"::/0",
		},
		ipnets: []*net.IPNet{
			{IP: net.IPv6zero, Mask: net.IPMask(net.IPv6zero)},
			{IP: net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, Mask: net.CIDRMask(0, 128)},
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("::/0"); return ipnet }(),
		},
	},
	{
		desc:   "IPv6, single IP",
		family: IPv6,
		strings: []string{
			"::1/128",
		},
		ipnets: []*net.IPNet{
			{IP: net.IPv6loopback, Mask: net.CIDRMask(128, 128)},
			{IP: net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01}, Mask: net.CIDRMask(128, 128)},
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("::1/128"); return ipnet }(),
		},
	},
	{
		desc: "IPv6 ifaddr (masked)",
		// This tests that if you try to parse an "ifaddr-style" CIDR string with
		// ParseCIDRSloppy, the *net.IPNet return value has the bits beyond the
		// prefix length masked out.
		family: IPv6,
		strings: []string{
			"2001:db8::/64",
			"2001:db8::1/64",
			"2001:db8::f00f:f0f0:0f0f:000f/64",
		},
		ipnets: []*net.IPNet{
			{IP: net.IP{0x20, 0x01, 0x0D, 0xB8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, Mask: net.CIDRMask(64, 128)},
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("2001:db8::/64"); return ipnet }(),
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("2001:db8::1/64"); return ipnet }(),
		},
	},
	{
		desc:   "IPv6 ifaddr",
		family: IPv6,
		strings: []string{
			"2001:db8::1/64",
		},
		ipnets: []*net.IPNet{
			{IP: net.IP{0x20, 0x01, 0x0D, 0xB8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, Mask: net.CIDRMask(64, 128)},
		},

		// The *net.IPNet return value of ParseCIDRSloppy() masks out the lower
		// bits, so the parsed version won't compare equal to .ipnets[0]
		skipParse: true,
	},
	{
		desc: "IPv4-mapped IPv6",
		// As in the IP tests, confirm that plain IPv4 and IPv4-mapped IPv6 are
		// treated as equivalent.
		family: IPv4,
		strings: []string{
			"1.1.1.0/24",
			"::ffff:1.1.1.0/120",
			"::ffff:01.01.01.00/0120",
		},
		ipnets: []*net.IPNet{
			{IP: net.IP{1, 1, 1, 0}, Mask: net.CIDRMask(24, 32)},
			{IP: net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xFF, 0xFF, 1, 1, 1, 0}, Mask: net.CIDRMask(120, 128)},
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("1.1.1.0/24"); return ipnet }(),
			func() *net.IPNet { _, ipnet, _ := net.ParseCIDR("::ffff:1.1.1.0/120"); return ipnet }(),

			// Explicitly test each of the 4 different combinations of 4-byte
			// or 16-byte IP and 4-byte or 16-byte Mask, all of which should
			// compare as equal and re-stringify to "1.1.1.0/24".
			{IP: net.IP{1, 1, 1, 0}, Mask: net.IPMask{255, 255, 255, 0}},
			{IP: net.IP{1, 1, 1, 0}, Mask: net.IPMask{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0}},
			{IP: net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xFF, 0xFF, 1, 1, 1, 0}, Mask: net.IPMask{255, 255, 255, 0}},
			{IP: net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xFF, 0xFF, 1, 1, 1, 0}, Mask: net.IPMask{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 0}},
		},
	},
}

// badTestCIDRs are bad test CIDR values. For each item:
//
// IPFamily tests (unless `skipFamily: true`):
//   - Each element of .strings should be identified as IPFamilyUnknown.
//   - Each element of .ipnets should be identified as IPFamilyUnknown.
//
// Parsing tests (unless `skipParse: true`):
//   - Each element of .strings should fail to parse.
//   - Each element of .ipnets should stringify to some error value that fails to parse.
var badTestCIDRs = []testCIDR{
	{
		desc: "empty string is not a CIDR",
		strings: []string{
			"",
		},
	},
	{
		desc: "random unparseable string is not a CIDR",
		strings: []string{
			"bad cidr",
		},
	},
	{
		desc: "CIDR with invalid IP is invalid",
		strings: []string{
			"1.2.300.0/24",
			"2001:db8000::/64",
		},
	},
	{
		desc: "CIDR with invalid prefix length is invalid",
		strings: []string{
			"1.2.3.4/64",
			"2001:db8::5/192",
			"1.2.3.0/-8",
			"1.2.3.0/+24",
		},
	},
	{
		desc: "URLs (that aren't also valid CIDRs) are invalid",
		strings: []string{
			"www.example.com/24",
			"192.168.0.1/0/99",
		},
	},
	{
		desc: "plain IP is not a CIDR",
		strings: []string{
			"1.2.3.4",
			"2001:db8::1",
		},
	},
	{
		desc: "CIDR with whitespace is invalid",
		strings: []string{
			" 1.2.3.0/24",
			"1.2.3.0/24 ",
		},
	},
	{
		desc: "nil is an invalid IPNet",
		ipnets: []*net.IPNet{
			nil,
		},
	},
	{
		desc: "IPNet containing invalid IP is invalid",
		ipnets: []*net.IPNet{
			{IP: net.IP{0x1}, Mask: net.CIDRMask(24, 32)},
		},
	},
	{
		desc: "IPNet containing non-CIDR Mask is invalid",
		ipnets: []*net.IPNet{
			{IP: net.IP{192, 168, 0, 0}, Mask: net.IPMask{255, 0, 255, 0}},
		},

		// IPFamilyOfCIDR only looks at IP and doesn't notice that Mask is invalid
		skipFamily: true,
	},
	{
		desc: "IPNet containing IPv6 IP and IPv4 Mask is invalid",
		ipnets: []*net.IPNet{
			{IP: net.IP{0x20, 0x01, 0x0D, 0xB8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, Mask: net.CIDRMask(24, 32)},
		},

		// IPFamilyOfCIDR only looks at IP and doesn't notice that Mask is invalid
		skipFamily: true,
	},
}

// TestGoodTestIPs confirms the Preconditions for goodTestIPs.
func TestGoodTestIPs(t *testing.T) {
	for _, tc := range goodTestIPs {
		t.Run(tc.desc, func(t *testing.T) {
			for i, ip := range tc.ips {
				if !ip.Equal(tc.ips[0]) {
					t.Errorf("BAD TEST DATA: IP %d %#v %q does not equal %#v %q", i+1, ip, ip, tc.ips[0], tc.ips[0])
				}
				str := ip.String()
				if str != tc.strings[0] {
					t.Errorf("BAD TEST DATA: IP %d %#v %q does not stringify to %q", i+1, ip, ip, tc.strings[0])
				}
			}
		})
	}
}

// TestGoodTestCIDRs confirms the Preconditions for goodTestCIDRs.
func TestGoodTestCIDRs(t *testing.T) {
	for _, tc := range goodTestCIDRs {
		t.Run(tc.desc, func(t *testing.T) {
			for i, ipnet := range tc.ipnets {
				if ipnet.String() != tc.strings[0] {
					t.Errorf("BAD TEST DATA: IPNet %d %#v %q does not stringify to %q", i+1, ipnet, ipnet, tc.strings[0])
				}
			}
		})
	}
}
