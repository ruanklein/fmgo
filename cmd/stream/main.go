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
	stream, err := client.Stream(context.Background(), fmgo.Request{Prompt: "Write a short story."})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()
	for stream.Next() {
		fmt.Print(stream.Text())
	}
	if err := stream.Err(); err != nil {
		log.Fatal(err)
	}
}
