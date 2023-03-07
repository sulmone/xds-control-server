package main

import (
	"context"
	"flag"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	v3server "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	log "github.com/sirupsen/logrus"
	"github.com/sulmone/xds-control-server/internal/processor"
	"github.com/sulmone/xds-control-server/internal/server"
	"github.com/sulmone/xds-control-server/internal/watcher"
)

var (
	l log.FieldLogger

	watchDirectoryFileName string
	port                   uint

	nodeID string
)

func init() {
	l = log.New()
	log.SetLevel(log.DebugLevel)

	// The port that this xDS server listens on
	flag.UintVar(&port, "port", 18000, "xDS management server port")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "xds-node", "Node ID")

	// Define the directory to watch for Envoy configuration files
	flag.StringVar(&watchDirectoryFileName, "watchDirectoryFileName", "config/service_mesh.yaml", "full path to directory to watch for files")
}

func main() {
	flag.Parse()

	// Create a cache
	cache := cache.NewSnapshotCache(true, cache.IDHash{}, l)

	log.Printf("Starting processor with nodeID: %s", nodeID)
	// Create a processor
	proc := processor.NewProcessor(context.Background(),
		cache, nodeID, log.WithField("context", "processor"))

	// Create initial snapshot from file
	proc.ProcessFile(watcher.NotifyMessage{
		Operation: watcher.Create,
		FilePath:  watchDirectoryFileName,
	})

	// Notify channel for file system events
	notifyCh := make(chan watcher.NotifyMessage)

	go func() {
		// Watch for file changes
		watcher.Watch(watchDirectoryFileName, notifyCh)
	}()

	go func() {
		// Run the xDS server
		ctx := context.Background()
		srv := v3server.NewServer(ctx, cache, nil)
		server.RunServer(ctx, srv, port)
	}()

	for {
		select {
		case msg := <-notifyCh:
			proc.ProcessFile(msg)
		}
	}
}
