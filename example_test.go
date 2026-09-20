// SPDX-License-Identifier: Apache-2.0

package fmgo_test

import (
	"context"
	"fmt"

	"github.com/ruanklein/fmgo"
)

func Example() {
	client, err := fmgo.New()
	if err != nil {
		return
	}
	response, err := client.Respond(context.Background(), fmgo.Request{
		Prompt:       "Explain goroutines.",
		Instructions: "Be concise.",
	})
	if err != nil {
		return
	}
	fmt.Println(response.Text)
}

func ExampleSchemaFor() {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	schema, err := fmgo.SchemaFor[Person]()
	if err != nil {
		return
	}
	_ = schema
}
