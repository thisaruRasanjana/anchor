package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	store := NewStore()

	walPath := os.Getenv("ANCHOR_WAL_PATH")
	if walPath == "" {
		walPath = "store.wal"
	}

	wal, err := NewWAL(walPath)
	if err != nil {
		panic(err)
	}
	defer wal.Close()

	if err := wal.Replay(store); err != nil {
		panic(err)
	}

	fmt.Println("Recovery complete")

	db := NewDatabase(store, wal)

	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		os.Interrupt,
		syscall.SIGTERM,
	)

	if err := StartServer(db, stop); err != nil {
		panic(err)
	}
}
