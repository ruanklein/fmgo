// SPDX-License-Identifier: Apache-2.0

package platform

import (
	"errors"
	"testing"
)

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

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		goos    string
		version string
		want    error
	}{
		{name: "supported", goos: "darwin", version: "27.0"},
		{name: "unsupported platform", goos: "linux", version: "27.0", want: ErrUnsupportedPlatform},
		{name: "unsupported version", goos: "darwin", version: "26.9", want: ErrUnsupportedVersion},
		{name: "invalid version", goos: "darwin", version: "unknown", want: ErrUnsupportedVersion},
	}
	for _, test := range tests {
		err := Validate(test.goos, test.version)
		if !errors.Is(err, test.want) {
			t.Fatalf("Validate(%q, %q) = %v, want %v", test.goos, test.version, err, test.want)
		}
	}
}
