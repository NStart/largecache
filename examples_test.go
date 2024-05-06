package largecache_test

import (
	"context"
	"fmt"
	"largecache"
	"log"
	"time"
)

func Example() {
	cache, _ := largecache.New(context.TODO(), largecache.DefaultConf(10*time.Minute))
	cache.Set("my-unique-key", []byte("value"))

	entry, _ := cache.Get("my-unique-key")
	fmt.Println(string(entry))
}

func Example_custom() {
	config := largecache.Config{
		Shards:             1024,
		LifeWindow:         10 * time.Minute,
		CleanWindow:        5 * time.Minute,
		MaxEntriesInWindow: 1000 * 10 * 60,
		MaxEntriesSize:     500,
		Verbose:            true,
		HardMaxCacheSize:   8192,
		OnRemove:           nil,
		OnRemoveWithReason: nil,
	}

	cache, initErr := largecache.New(context.Background(), config)
	if initErr != nil {
		log.Fatal(initErr)
	}

	err := cache.Set("my-unique-key", []byte("value"))
	if err != nil {
		log.Fatal(err)
	}

	entry, err := cache.Get("my-unique-key")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(entry))

}
