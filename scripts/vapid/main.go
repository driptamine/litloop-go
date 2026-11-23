// Command vapid generates a VAPID key pair for web push notifications.
// Usage: go run ./scripts/vapid
package main

import (
	"fmt"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	private, public, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		panic(err)
	}
	fmt.Println("VAPID_PUBLIC_KEY=" + public)
	fmt.Println("VAPID_PRIVATE_KEY=" + private)
}
