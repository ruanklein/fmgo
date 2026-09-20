// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ruanklein/fmgo/v1"
)

func main() {
	response, err := fmgo.New().Respond(context.Background(), fmgo.Request{
		Prompt:       "Explain goroutines.",
		Instructions: "Be concise.",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(response.Text)
}
