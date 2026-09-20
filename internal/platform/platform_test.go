// SPDX-License-Identifier: Apache-2.0

package platform

import "testing"

func TestMajorVersion(t *testing.T) {
	for _, test := range []struct {
		version string
		want    int
		valid   bool
	}{
		{version: "27.0", want: 27, valid: true},
		{version: "28", want: 28, valid: true},
		{version: "invalid", valid: false},
	} {
		got, err := MajorVersion(test.version)
		if (err == nil) != test.valid || got != test.want {
			t.Fatalf("MajorVersion(%q) = %d, %v", test.version, got, err)
		}
	}
}
