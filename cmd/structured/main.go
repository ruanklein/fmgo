// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ruanklein/fmgo/v1"
)

type person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	client, err := fmgo.New()
	if err != nil {
		log.Fatal(err)
	}
	value, err := fmgo.RespondAs[person](context.Background(), client, fmgo.Request{
		Prompt: "Generate a fictional person.",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s is %d\n", value.Name, value.Age)
}
