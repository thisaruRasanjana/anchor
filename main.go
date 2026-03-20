package main

import "fmt"

func main() {
	store := NewStore()

	store.Set("name", "Thisaru")

	value, ok := store.Get("name")
	fmt.Println("Before delete:", value, ok)

	store.Delete("name")

	value, ok = store.Get("name")
	fmt.Println("After delete:", value, ok)
}