// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ruanklein/fmgo"
)

func main() {
	client, err := fmgo.New()
	if err != nil {
		log.Fatal(err)
	}
	response, err := client.Respond(context.Background(), fmgo.Request{
		Prompt:       "Explain goroutines.",
		Instructions: "Be concise.",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(response.Text)
}
