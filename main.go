package main

import (
	"fmt"
	"time"
)

func main() {
	store := NewStore()

store.SetWithTTL("name", "Thisaru", 3)

fmt.Println("Waiting 5 seconds...")
time.Sleep(5 * time.Second)

value, ok := store.Get("name")
fmt.Println(value, ok)
}
