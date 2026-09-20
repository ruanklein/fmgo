// SPDX-License-Identifier: Apache-2.0

// Package fmgo provides a Go interface to Apple's Foundation Models CLI.
package fmgo

import (
	"context"
	"errors"
	"os/exec"

	"github.com/ruanklein/fmgo/v1/internal/platform"
)

func init() {
	platform.Require()
}

// Client invokes the native fm executable.
type Client struct {
	executable string
}

// Option configures a Client.
type Option func(*Client)

// WithExecutable overrides the fm executable path.
func WithExecutable(path string) Option {
	return func(client *Client) {
		client.executable = path
	}
}

// New creates a Client that uses fm from PATH by default.
func New(options ...Option) *Client {
	client := &Client{executable: "fm"}
	for _, option := range options {
		option(client)
	}
	return client
}

func (c *Client) executablePath() (string, error) {
	path, err := exec.LookPath(c.executable)
	if err != nil {
		return "", ErrFMNotFound
	}
	return path, nil
}

func (c *Client) run(ctx context.Context, args ...string) ([]byte, string, error) {
	path, err := c.executablePath()
	if err != nil {
		return nil, "", err
	}
	result, err := runCommand(ctx, path, args...)
	if err != nil {
		return result.stdout, result.stderr, commandError(args, result.stderr, err)
	}
	return result.stdout, result.stderr, nil
}

func isNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound)
}
