package main

import (
	"fmt"
	"time"
)

func main() {
	store := NewStore()

	store.SetWithTTL("name", "Thisaru", 5)

	value, ok := store.Get("name")
	fmt.Println("Immediately:", value, ok)

	time.Sleep(6 * time.Second)

	value, ok = store.Get("name")
	fmt.Println("After 6 seconds:", value, ok)
}
