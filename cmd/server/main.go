package main

import (
	"context"
	"flag"
	"sync"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	v3discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
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

type logger struct{}

func (logger logger) Infof(format string, args ...interface{}) {
	l.Infof(format, args...)
}
func (logger logger) Errorf(format string, args ...interface{}) {
	l.Errorf(format, args...)
}
func (cb *callbacks) Report() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	l.WithFields(log.Fields{"fetches": cb.fetches, "requests": cb.requests}).Info("cb.Report()  callbacks")
}
func (cb *callbacks) OnStreamOpen(ctx context.Context, id int64, typ string) error {
	l.Infof("OnStreamOpen %d open for Type [%s]", id, typ)
	return nil
}
func (cb *callbacks) OnStreamClosed(id int64, node *core.Node) {
	l.Infof("OnStreamClosed nodeId[%s] %d closed", node.Id, id)
}
func (cb *callbacks) OnStreamRequest(id int64, r *v3discovery.DiscoveryRequest) error {
	l.Infof("OnStreamRequest %d  Request[%v]", id, r.TypeUrl)
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.requests++
	if cb.signal != nil {
		close(cb.signal)
		cb.signal = nil
	}
	return nil
}
func (cb *callbacks) OnStreamResponse(ctx context.Context, id int64, req *v3discovery.DiscoveryRequest, resp *v3discovery.DiscoveryResponse) {
	l.Infoln(resp.GetResources())
	l.Infof("OnStreamResponse... %d   Request [%v],  Response[%v]", id, req.TypeUrl, resp.TypeUrl)
	cb.Report()
}
func (cb *callbacks) OnFetchRequest(ctx context.Context, req *v3discovery.DiscoveryRequest) error {
	l.Infof("OnFetchRequest... Request [%v]", req.TypeUrl)
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.fetches++
	if cb.signal != nil {
		close(cb.signal)
		cb.signal = nil
	}
	return nil
}
func (cb *callbacks) OnFetchResponse(req *v3discovery.DiscoveryRequest, resp *v3discovery.DiscoveryResponse) {
	l.Infof("OnFetchResponse... Resquest[%v],  Response[%v]", req.TypeUrl, resp.TypeUrl)
}

func (cb *callbacks) OnDeltaStreamClosed(id int64, node *core.Node) {
	l.Infof("OnDeltaStreamClosed nodeId[%s]... %v", node.Id, id)
}

func (cb *callbacks) OnDeltaStreamOpen(ctx context.Context, id int64, typ string) error {
	l.Infof("OnDeltaStreamOpen... %v  of type %s", id, typ)
	return nil
}

func (c *callbacks) OnStreamDeltaRequest(i int64, request *v3discovery.DeltaDiscoveryRequest) error {
	l.Infof("OnStreamDeltaRequest... %v  of type %s", i, request)
	return nil
}

func (c *callbacks) OnStreamDeltaResponse(i int64, request *v3discovery.DeltaDiscoveryRequest, response *v3discovery.DeltaDiscoveryResponse) {
	l.Infof("OnStreamDeltaResponse... %v  of type %s", i, request)
}

type callbacks struct {
	signal   chan struct{}
	fetches  int
	requests int
	mu       sync.Mutex
}

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
		signal := make(chan struct{})
		cb := &callbacks{
			signal:   signal,
			fetches:  0,
			requests: 0,
		}
		// Run the xDS server
		ctx := context.Background()
		srv := v3server.NewServer(ctx, cache, cb)
		server.RunServer(ctx, srv, port)
	}()

	for {
		select {
		case msg := <-notifyCh:
			proc.ProcessFile(msg)
		}
	}
}
