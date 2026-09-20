// SPDX-License-Identifier: Apache-2.0

package platform

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

const MinimumMajorVersion = 27

// Require verifies that the current machine can run fmgo.
func Require() {
	if runtime.GOOS != "darwin" {
		panic("fmgo: unsupported platform: fmgo requires macOS 27 or later")
	}

	output, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		panic("fmgo: unsupported platform: unable to determine macOS version; fmgo requires macOS 27 or later")
	}

	major, err := MajorVersion(string(output))
	if err != nil || major < MinimumMajorVersion {
		panic("fmgo: unsupported platform: fmgo requires macOS 27 or later")
	}
}

// MajorVersion extracts the major component of a macOS product version.
func MajorVersion(version string) (int, error) {
	majorText, _, _ := strings.Cut(strings.TrimSpace(version), ".")
	major, err := strconv.Atoi(majorText)
	if err != nil || major < 0 {
		return 0, fmt.Errorf("invalid macOS product version %q", version)
	}
	return major, nil
}
