package main

import (
	"fmt"
)

func main() {
	store := NewStore()

	err := StartServer(store)
	if err != nil {
		fmt.Println(err)
	}
}
