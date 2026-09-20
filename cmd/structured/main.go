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
	value, err := fmgo.RespondAs[person](context.Background(), fmgo.New(), fmgo.Request{
		Prompt: "Generate a fictional person.",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s is %d\n", value.Name, value.Age)
}
