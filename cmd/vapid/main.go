package main

import (
	"fmt"
	"log"

	"github.com/danielradosa/nikkibase/push"
)

func main() {
	private, public, err := push.NewVAPIDKey()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("VAPID_PRIVATE_KEY=%s\nVAPID_PUBLIC_KEY=%s\n", private, public)
}
