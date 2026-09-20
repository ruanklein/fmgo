// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"

	"github.com/ruanklein/fmgo/v1"
)

func main() {
	server, err := fmgo.New().Serve(context.Background(), fmgo.ServerOptions{Port: 8080})
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()
	log.Printf("fm server listening on %s", server.Addr())
	if err := server.Wait(); err != nil {
		log.Fatal(err)
	}
}
