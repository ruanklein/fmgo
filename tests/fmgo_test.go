// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ruanklein/fmgo/v1"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("FMGO_HELPER") != "1" {
		return
	}
	args := os.Args[1:]
	if len(args) == 0 {
		os.Exit(2)
	}
	switch args[0] {
	case "respond":
		if contains(args, "--stream") {
			fmt.Fprint(os.Stdout, "hello")
			os.Exit(0)
		}
		if contains(args, "fail") {
			fmt.Fprint(os.Stderr, "model is not available")
			os.Exit(7)
		}
		fmt.Fprint(os.Stdout, strings.Join(args, "|"))
	case "count-tokens":
		fmt.Fprint(os.Stdout, "42\n")
	case "available":
		fmt.Fprint(os.Stdout, "System model unavailable: modelNotReady\n")
	case "license":
		fmt.Fprint(os.Stdout, "Agreed to license FM1 version 1.0 on today.\n")
	default:
		os.Exit(2)
	}
	os.Exit(0)
}

func TestRespondBuildsArgumentsWithoutShellInterpolation(t *testing.T) {
	client := fakeClient(t)
	response, err := client.Respond(context.Background(), fmgo.Request{
		Prompt:       `hello "world"; $(nope)`,
		Instructions: "be concise",
		Images:       []string{"photo one.png"},
		ImageLabels:  []string{"source"},
		Text:         []string{"more text"},
		Tools:        []fmgo.Tool{fmgo.ToolOCR},
		Model:        fmgo.ModelSystem,
		Greedy:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--no-stream", "--instructions|be concise", "--image|photo one.png", `hello "world"; $(nope)`} {
		if !strings.Contains(response.Text, want) {
			t.Fatalf("response %q does not contain %q", response.Text, want)
		}
	}
}

func TestStream(t *testing.T) {
	stream, err := fakeClient(t).Stream(context.Background(), fmgo.Request{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if !stream.Next() || stream.Text() != "hello" {
		t.Fatalf("unexpected stream text %q", stream.Text())
	}
	if stream.Next() || stream.Err() != nil {
		t.Fatalf("unexpected stream completion: %v", stream.Err())
	}
}

func TestTypedErrorsAndParsing(t *testing.T) {
	client := fakeClient(t)
	_, err := client.Respond(context.Background(), fmgo.Request{Prompt: "fail"})
	if !errors.Is(err, fmgo.ErrModelUnavailable) {
		t.Fatalf("expected model error, got %v", err)
	}
	count, err := client.CountTokens(context.Background(), fmgo.TokenRequest{Prompt: "hello"})
	if err != nil || count != 42 {
		t.Fatalf("count %d, err %v", count, err)
	}
	availability, err := client.Available(context.Background())
	if err != nil || availability.Models[0].Reason != "modelNotReady" {
		t.Fatalf("availability %#v, err %v", availability, err)
	}
	license, err := client.License(context.Background())
	if err != nil || !license.Accepted {
		t.Fatalf("license %#v, err %v", license, err)
	}
}

func TestSchemaFor(t *testing.T) {
	type Address struct {
		City string `json:"city"`
	}
	type Person struct {
		Name    string   `json:"name"`
		Address Address  `json:"address"`
		Tags    []string `json:"tags,omitempty"`
		Ignored string   `json:"-"`
	}
	schema, err := fmgo.SchemaFor[Person]()
	if err != nil {
		t.Fatal(err)
	}
	text := string(schema)
	for _, want := range []string{`"name"`, `"address"`, `"city"`, `"tags"`, `"required":["name","address"]`} {
		if !strings.Contains(text, want) {
			t.Fatalf("schema %s does not contain %s", text, want)
		}
	}
}

func TestSchemaRejectsRecursiveAndUnsupportedTypes(t *testing.T) {
	type Recursive struct{ Next *Recursive }
	if _, err := fmgo.SchemaFor[Recursive](); err == nil {
		t.Fatal("expected recursive type error")
	}
	if _, err := fmgo.SchemaFor[func()](); err == nil {
		t.Fatal("expected function type error")
	}
}

func fakeClient(t *testing.T) *fmgo.Client {
	t.Helper()
	t.Setenv("FMGO_HELPER", "1")
	return fmgo.New(fmgo.WithExecutable(os.Args[0]))
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
