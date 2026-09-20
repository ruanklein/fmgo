// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ruanklein/fmgo"
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
		if contains(args, "block") {
			select {}
		}
		if contains(args, "--stream") {
			fmt.Fprint(os.Stdout, "hello")
			os.Exit(0)
		}
		if contains(args, "license-required") {
			fmt.Fprint(os.Stderr, "license not accepted")
			os.Exit(69)
		}
		if contains(args, "fail") {
			fmt.Fprint(os.Stderr, "model is not available")
			os.Exit(7)
		}
		if os.Getenv("FMGO_HELPER_STRUCTURED_RESPONSE") == "1" {
			fmt.Fprint(os.Stdout, `{"name":"Ada"}`)
			os.Exit(0)
		}
		fmt.Fprint(os.Stdout, strings.Join(args, "|"))
	case "count-tokens":
		if contains(args, "invalid") {
			fmt.Fprint(os.Stdout, "not-a-number\n")
			os.Exit(0)
		}
		fmt.Fprint(os.Stdout, "42\n")
	case "available":
		if contains(args, "invalid") {
			fmt.Fprint(os.Stdout, "unexpected")
			os.Exit(0)
		}
		fmt.Fprint(os.Stdout, "System model unavailable: modelNotReady\n")
	case "schema":
		fmt.Fprint(os.Stdout, `{"type":"object","properties":{"name":{"type":"string"}}}`)
	case "license":
		fmt.Fprint(os.Stdout, "Agreed to license FM1 version 1.0 on today.\n")
	case "chat":
		if os.Getenv("FMGO_HELPER_CHAT_ECHO") == "1" {
			fmt.Fprint(os.Stdout, strings.Join(args, "|"))
			os.Exit(0)
		}
		select {}
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

func TestRespondBuildsOptionalArguments(t *testing.T) {
	response, err := fakeClient(t).Respond(context.Background(), fmgo.Request{
		Prompt:         "hello",
		SchemaFile:     "schema.json",
		Resume:         "resume.json",
		SaveTranscript: "transcript.json",
		Verbose:        true,
		UseCase:        fmgo.UseCaseContentTagging,
		Guardrails:     fmgo.GuardrailPermissiveContentTransformations,
		Tools:          []fmgo.Tool{fmgo.ToolBarcode},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"--schema|schema.json",
		"--resume|resume.json",
		"--save-transcript|transcript.json",
		"--verbose",
		"--use-case|content-tagging",
		"--guardrails|permissive-content-transformations",
		"--tool|barcode",
	} {
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
	if _, err := client.ShowLicense(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CountTokens(context.Background(), fmgo.TokenRequest{Prompt: "invalid"}); !errors.Is(err, fmgo.ErrUnexpectedOutput) {
		t.Fatalf("expected unexpected output error, got %v", err)
	}
}

func TestRespondAsAndGenerateSchema(t *testing.T) {
	type person struct {
		Name string `json:"name"`
	}

	t.Setenv("FMGO_HELPER_STRUCTURED_RESPONSE", "1")
	client := fakeClient(t)
	schema, err := client.GenerateSchema(context.Background(), fmgo.ObjectSchema{
		Name: "Person",
		Properties: []fmgo.SchemaProperty{
			{Name: "name", Type: fmgo.SchemaString},
			{Name: "age", Type: fmgo.SchemaInteger, Optional: true},
			{Name: "score", Type: fmgo.SchemaDouble},
			{Name: "active", Type: fmgo.SchemaBoolean},
			{Name: "address", Type: fmgo.SchemaObject, Schema: fmgo.Schema(`{}`)},
			{Name: "result", Type: fmgo.SchemaAnyOf, Choices: []fmgo.Schema{fmgo.Schema(`{}`)}},
		},
	})
	if err != nil || len(schema) == 0 {
		t.Fatalf("schema %s, err %v", schema, err)
	}

	value, err := fmgo.RespondAs[person](context.Background(), client, fmgo.Request{Prompt: "person"})
	if err != nil {
		t.Fatal(err)
	}
	if value.Name != "Ada" {
		t.Fatalf("name = %q, want Ada", value.Name)
	}
}

func TestTokenRequestAndSchemaHelpers(t *testing.T) {
	count, err := fakeClient(t).CountTokens(context.Background(), fmgo.TokenRequest{
		Prompt:       "hello",
		Instructions: "be concise",
		Text:         []string{"additional text"},
		Images:       []string{"image.png"},
		Transcript:   "transcript.json",
	})
	if err != nil || count != 42 {
		t.Fatalf("count %d, err %v", count, err)
	}

	if _, err := fmgo.ParseSchema([]byte(`{"type":"object"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := fmgo.ParseSchema([]byte(`not-json`)); err == nil {
		t.Fatal("expected invalid schema error")
	}
	type record struct {
		Name string `json:"name"`
	}
	if _, err := fmgo.SchemaFrom(record{}); err != nil {
		t.Fatal(err)
	}
}

func TestRequestValidationAndTypedErrors(t *testing.T) {
	client := fakeClient(t)
	_, err := client.Respond(context.Background(), fmgo.Request{
		Images:      []string{"photo.png"},
		ImageLabels: []string{"first", "second"},
	})
	if err == nil {
		t.Fatal("expected image label validation error")
	}
	_, err = client.Respond(context.Background(), fmgo.Request{
		Schema:     fmgo.Schema(`{}`),
		SchemaFile: "schema.json",
	})
	if err == nil {
		t.Fatal("expected schema conflict error")
	}
	_, err = client.Respond(context.Background(), fmgo.Request{Prompt: "license-required"})
	if !errors.Is(err, fmgo.ErrLicenseRequired) {
		t.Fatalf("expected license error, got %v", err)
	}
	var commandError *fmgo.CommandError
	if !errors.As(err, &commandError) || commandError.ExitCode != 69 {
		t.Fatalf("expected command error with exit code 69, got %v", err)
	}
	missingClient, err := fmgo.New(fmgo.WithExecutable(filepath.Join(t.TempDir(), "missing-fm")))
	if err != nil {
		t.Fatal(err)
	}
	_, err = missingClient.Respond(context.Background(), fmgo.Request{Prompt: "hello"})
	if !errors.Is(err, fmgo.ErrFMNotFound) {
		t.Fatalf("expected missing executable error, got %v", err)
	}
}

func TestStreamCancellationAndChatValidation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stream, err := fakeClient(t).Stream(ctx, fmgo.Request{Prompt: "block"})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	defer stream.Close()
	if stream.Next() {
		t.Fatal("stream produced output after cancellation")
	}
	if stream.Err() == nil {
		t.Fatal("expected stream cancellation error")
	}

	_, err = fakeClient(t).StartChat(context.Background(), fmgo.ChatOptions{
		Continue: true,
		Resume:   "session",
	})
	if err == nil {
		t.Fatal("expected conflicting chat option error")
	}
}

func TestChatArguments(t *testing.T) {
	t.Setenv("FMGO_HELPER_CHAT_ECHO", "1")
	session, err := fakeClient(t).StartChat(context.Background(), fmgo.ChatOptions{
		Model:           fmgo.ModelSystem,
		Instructions:    "be concise",
		Resume:          "session",
		SetDefaultModel: fmgo.ModelSystem,
		Tools:           []fmgo.Tool{fmgo.ToolBarcode, fmgo.ToolOCR},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	output, err := io.ReadAll(session.Stdout())
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Wait(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"--model|system",
		"--instructions|be concise",
		"--resume|session",
		"--set-default-model|system",
		"--tool|barcode",
		"--tool|ocr",
	} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("chat output %q does not contain %q", output, want)
		}
	}

	continued, err := fakeClient(t).StartChat(context.Background(), fmgo.ChatOptions{Continue: true})
	if err != nil {
		t.Fatal(err)
	}
	defer continued.Close()
	continuedOutput, err := io.ReadAll(continued.Stdout())
	if err != nil {
		t.Fatal(err)
	}
	if err := continued.Wait(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(continuedOutput), "--continue") {
		t.Fatalf("chat output %q does not contain --continue", continuedOutput)
	}
}

func TestServerValidation(t *testing.T) {
	client := fakeClient(t)
	_, err := client.Serve(context.Background(), fmgo.ServerOptions{Port: 0})
	if err == nil {
		t.Fatal("expected invalid port error")
	}
	_, err = client.Serve(context.Background(), fmgo.ServerOptions{
		Host:   "127.0.0.1",
		Port:   8080,
		Socket: "fm.sock",
	})
	if err == nil {
		t.Fatal("expected conflicting socket error")
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
	for _, want := range []string{
		`"title":"Person"`,
		`"additionalProperties":false`,
		`"name"`,
		`"address"`,
		`"city"`,
		`"tags"`,
		`"required":["name","address"]`,
	} {
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
	client, err := fmgo.New(fmgo.WithExecutable(os.Args[0]))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
