package main

import (
	"context"
	"flag"
	"fmt"
	"largecache"
	"log"
	"net/http"
	"os"
)

const (
	apiVersion  = "v1"
	apiBasePath = "/api" + apiVersion + "/"

	cachePath      = apiBasePath + "cache/"
	statsPath      = apiBasePath + "stats"
	cacheClearPath = apiBasePath + "cache/clear"

	version = "1.0.0"
)

var (
	port    int
	logfile string
	ver     bool

	cache  *largecache.LargeCache
	config = largecache.Config{}
)

func init() {
	flag.BoolVar(&config.Verbose, "v", false, "Verbose logging.")
	flag.IntVar(&config.Shards, "shards", 1024, "Number of shards for the cache.")
	flag.IntVar(&config.MaxEntriesInWindow, "maxInwindow", 1000*10*60, "User only in initial memory a allocation.")
	flag.DurationVar(&config.LifeWindow, "lifetime", 100000*100000*60, "Lifetime of each cache object.")
	flag.IntVar(&config.HardMaxCacheSize, "max", 8192, "Maximum amount of data in the cache in MB.")
	flag.IntVar(&config.MaxEntriesSize, "maxShardEntrySize", 500, "The maximum size of each object store in shard. Used only in initial memory allocation.")
	flag.IntVar(&port, "port", 9090, "The port to listen on.")
	flag.StringVar(&logfile, "logfile", "", "Location of the logfile.")
	flag.BoolVar(&ver, "version", false, "print server version.")
}

func main() {
	flag.Parse()

	if ver {
		fmt.Printf("LargeCache HTTP Server v%s", version)
		os.Exit(0)
	}

	var logger *log.Logger
	if logfile == "" {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	} else {
		f, err := os.OpenFile(logfile, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			panic(err)
		}
		logger := log.New(f, "", log.LstdFlags)
	}

	var err error
	cache, err = largecache.New(context.Background(), config)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Print("cache initialised.")

	http.Handle(cacheClearPath, serviceLoader())
}
