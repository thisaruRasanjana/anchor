package main

import "fmt"

func main() {
	store := NewStore()

	wal, err := NewWAL("store.wal")
	if err != nil {
		panic(err)
	}
	defer wal.Close()

	if err := wal.Replay(store); err != nil {
		panic(err)
	}

	fmt.Println("Recovery complete")

	db := NewDatabase(store, wal)

	if err := StartServer(db); err != nil {
		panic(err)
	}
}
