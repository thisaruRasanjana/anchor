package main

import (
	"fmt"
	"sync"
)

func main() {
	store := NewStore()
	var wg sync.WaitGroup

	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			value := fmt.Sprintf("value%d", i)
			store.SetWithTTL(key, value, 5)
			_, ok := store.Get(key)
			if !ok {
				fmt.Printf("ERROR: %s not found\n", key)
			}
		}(i)
	}
	wg.Wait()
	fmt.Println("All workers finished successfully.")
}
