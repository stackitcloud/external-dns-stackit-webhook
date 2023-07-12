package main

import "github.com/stackitcloud/external-dns-stackit-webhook/cmd/webhook/cmd"

func main() {
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
