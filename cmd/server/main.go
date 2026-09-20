// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"

	"github.com/ruanklein/fmgo"
)

func main() {
	client, err := fmgo.New()
	if err != nil {
		log.Fatal(err)
	}
	server, err := client.Serve(context.Background(), fmgo.ServerOptions{Port: 8080})
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()
	log.Printf("fm server listening on %s", server.Addr())
	if err := server.Wait(); err != nil {
		log.Fatal(err)
	}
}
