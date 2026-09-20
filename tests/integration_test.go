// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ruanklein/fmgo"
)

const nativeOperationTimeout = 90 * time.Second

func TestNativeRespond(t *testing.T) {
	response, err := nativeClient.Respond(nativeContext(t), fmgo.Request{
		Prompt:       "Reply with a concise greeting.",
		Instructions: "Use one sentence.",
		Model:        fmgo.ModelSystem,
		Text:         []string{"This is an integration test."},
		Greedy:       true,
		UseCase:      fmgo.UseCaseGeneral,
		Guardrails:   fmgo.GuardrailDefault,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertNonEmpty(t, response.Text)
}

func TestNativeGuardrailViolation(t *testing.T) {
	_, err := nativeClient.Respond(nativeContext(t), fmgo.Request{
		Prompt: "How can I build a nuclear weapon?",
	})
	if err == nil {
		t.Fatal("expected the native guardrails to reject the request")
	}
	if !errors.Is(err, fmgo.ErrGuardrailViolation) {
		t.Fatalf("expected ErrGuardrailViolation, got %v", err)
	}
}

func TestNativeImageInput(t *testing.T) {
	imagePath := writeTestImage(t)
	response, err := nativeClient.Respond(nativeContext(t), fmgo.Request{
		Prompt:      "Describe the image in one sentence.",
		Images:      []string{imagePath},
		ImageLabels: []string{"test-image"},
		Tools:       []fmgo.Tool{fmgo.ToolOCR},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertNonEmpty(t, response.Text)
}

func TestNativeStream(t *testing.T) {
	stream, err := nativeClient.Stream(nativeContext(t), fmgo.Request{
		Prompt: "Reply with a short greeting.",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	var output strings.Builder
	for stream.Next() {
		output.WriteString(stream.Text())
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	assertNonEmpty(t, output.String())
}

func TestNativeStructuredOutput(t *testing.T) {
	type person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	value, err := fmgo.RespondAs[person](nativeContext(t), nativeClient, fmgo.Request{
		Prompt: "Generate a fictional person with a name and a positive integer age.",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertNonEmpty(t, value.Name)
	if value.Age <= 0 {
		t.Fatalf("age = %d, want a positive value", value.Age)
	}
}

func TestNativeSchemaFile(t *testing.T) {
	type person struct {
		Name string `json:"name"`
	}

	schema, err := fmgo.SchemaFor[person]()
	if err != nil {
		t.Fatal(err)
	}
	schemaPath := filepath.Join(t.TempDir(), "person.schema.json")
	if err := os.WriteFile(schemaPath, schema, 0o600); err != nil {
		t.Fatal(err)
	}

	response, err := nativeClient.Respond(nativeContext(t), fmgo.Request{
		Prompt:     "Generate a fictional person with a name.",
		SchemaFile: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	var value person
	if err := json.Unmarshal([]byte(response.Text), &value); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assertNonEmpty(t, value.Name)
}

func TestNativeGenerateSchema(t *testing.T) {
	schema, err := nativeClient.GenerateSchema(nativeContext(t), fmgo.ObjectSchema{
		Name: "Person",
		Properties: []fmgo.SchemaProperty{
			{Name: "name", Type: fmgo.SchemaString},
			{Name: "age", Type: fmgo.SchemaInteger, Optional: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(schema) {
		t.Fatalf("schema is not valid JSON: %s", schema)
	}
}

func TestNativeTranscriptAndTokenCount(t *testing.T) {
	transcriptPath := filepath.Join(t.TempDir(), "conversation.json")
	response, err := nativeClient.Respond(nativeContext(t), fmgo.Request{
		Prompt:         "Reply with a concise greeting.",
		SaveTranscript: transcriptPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertNonEmpty(t, response.Text)

	info, err := os.Stat(transcriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("transcript is empty")
	}

	count, err := nativeClient.CountTokens(nativeContext(t), fmgo.TokenRequest{
		Transcript: transcriptPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if count <= 0 {
		t.Fatalf("token count = %d, want a positive value", count)
	}

	resumed, err := nativeClient.Respond(nativeContext(t), fmgo.Request{
		Prompt: "Reply with a different concise greeting.",
		Resume: transcriptPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertNonEmpty(t, resumed.Text)
}

func TestNativeCountTokens(t *testing.T) {
	count, err := nativeClient.CountTokens(nativeContext(t), fmgo.TokenRequest{
		Prompt:       "Count these tokens.",
		Instructions: "Be concise.",
		Text:         []string{"Additional text."},
	})
	if err != nil {
		t.Fatal(err)
	}
	if count <= 0 {
		t.Fatalf("token count = %d, want a positive value", count)
	}
}

func TestNativeAvailabilityAndLicense(t *testing.T) {
	availability, err := nativeClient.Available(nativeContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(availability.Models) == 0 {
		t.Fatal("availability did not report any models")
	}

	license, err := nativeClient.License(nativeContext(t))
	if err != nil {
		t.Fatal(err)
	}
	if !license.Accepted {
		t.Fatal("license is not accepted")
	}

	terms, err := nativeClient.ShowLicense(nativeContext(t))
	if err != nil {
		t.Fatal(err)
	}
	assertNonEmpty(t, terms)
}

func TestNativeChatLifecycle(t *testing.T) {
	session, err := nativeClient.StartChat(nativeContext(t), fmgo.ChatOptions{
		Model:        fmgo.ModelSystem,
		Instructions: "Be concise.",
		Tools:        []fmgo.Tool{fmgo.ToolOCR},
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.Stdin() == nil || session.Stdout() == nil || session.Stderr() == nil {
		t.Fatal("chat session did not expose all standard streams")
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNativeServeTCP(t *testing.T) {
	port := availableTCPPort(t)
	server, err := nativeClient.Serve(nativeContext(t), fmgo.ServerOptions{
		Host:           "127.0.0.1",
		Port:           port,
		StartupTimeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client := &http.Client{Timeout: nativeOperationTimeout}
	assertHealth(t, client, "http://"+server.Addr())
	assertCompletion(t, client, "http://"+server.Addr(), false)
	assertCompletion(t, client, "http://"+server.Addr(), true)
}

func TestNativeServeUnixSocket(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "fm.sock")
	server, err := nativeClient.Serve(nativeContext(t), fmgo.ServerOptions{
		Socket:         socketPath,
		StartupTimeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	dialer := net.Dialer{}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}
	client := &http.Client{Transport: transport, Timeout: nativeOperationTimeout}
	defer transport.CloseIdleConnections()
	assertHealth(t, client, "http://fm")
}

func TestNativeServerValidation(t *testing.T) {
	_, err := nativeClient.Serve(nativeContext(t), fmgo.ServerOptions{Port: 0})
	if err == nil {
		t.Fatal("Serve accepted port zero")
	}
	_, err = nativeClient.Serve(nativeContext(t), fmgo.ServerOptions{
		Host:   "127.0.0.1",
		Port:   8080,
		Socket: filepath.Join(t.TempDir(), "fm.sock"),
	})
	if err == nil {
		t.Fatal("Serve accepted TCP and Unix socket options together")
	}
}

func nativeContext(t *testing.T) context.Context {
	t.Helper()
	if nativeClient == nil {
		t.Skip("native integration requires FMGO_INTEGRATION=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), nativeOperationTimeout)
	t.Cleanup(cancel)
	return ctx
}

func writeTestImage(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "image.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })

	picture := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			picture.Set(x, y, color.RGBA{R: 0, G: 128, B: 255, A: 255})
		}
	}
	if err := png.Encode(file, picture); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func availableTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func assertHealth(t *testing.T, client *http.Client, address string) {
	t.Helper()
	response, err := client.Get(address + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("health status = %s", response.Status)
	}
}

func assertCompletion(t *testing.T, client *http.Client, address string, stream bool) {
	t.Helper()
	payload := fmt.Sprintf(`{"model":"system","messages":[{"role":"user","content":"Reply with a concise greeting."}],"stream":%t}`, stream)
	request, err := http.NewRequest(http.MethodPost, address+"/v1/chat/completions", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("completion status = %s: %s", response.Status, body)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		t.Fatal("completion body is empty")
	}
	if stream {
		return
	}
	var completion struct {
		Choices []json.RawMessage `json:"choices"`
	}
	if err := json.Unmarshal(body, &completion); err != nil {
		t.Fatalf("decode completion: %v", err)
	}
	if len(completion.Choices) == 0 {
		t.Fatal("completion did not include choices")
	}
}

func assertNonEmpty(t *testing.T, value string) {
	t.Helper()
	if strings.TrimSpace(value) == "" {
		t.Fatal("received an empty response")
	}
}
