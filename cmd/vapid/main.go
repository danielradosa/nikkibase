package main

import (
	"fmt"
	"log"

	"github.com/danielradosa/nikkibase/push"
)

func main() {
	_, _, err := push.NewVAPIDKey()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ VAPID keys generated successfully.") 
}
