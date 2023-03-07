//   Copyright Steve Sloka 2021
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package processor

import (
	"context"
	"math"
	"math/rand"
	"os"
	"reflect"
	"strconv"

	"github.com/sulmone/xds-control-server/internal/resources"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"

	"github.com/sulmone/xds-control-server/internal/xdscache"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	v3resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/sirupsen/logrus"
	"github.com/sulmone/xds-control-server/internal/watcher"
)

type Processor struct {
	ctx    context.Context
	cache  cache.SnapshotCache
	nodeID string

	// snapshotVersion holds the current version of the snapshot.
	snapshotVersion int64

	logrus.FieldLogger

	xdsCache xdscache.XDSCache
}

func NewProcessor(ctx context.Context, cache cache.SnapshotCache, nodeID string, log logrus.FieldLogger) *Processor {
	return &Processor{
		ctx:             ctx,
		cache:           cache,
		nodeID:          nodeID,
		snapshotVersion: rand.Int63n(1000),
		FieldLogger:     log,
		xdsCache: xdscache.XDSCache{
			Services: make(map[string]resources.Service),
		},
	}
}

// newSnapshotVersion increments the current snapshotVersion
// and returns as a string.
func (p *Processor) newSnapshotVersion() string {

	// Reset the snapshotVersion if it ever hits max size.
	if p.snapshotVersion == math.MaxInt64 {
		p.snapshotVersion = 0
	}

	// Increment the snapshot version & return as string.
	p.snapshotVersion++
	return strconv.FormatInt(p.snapshotVersion, 10)
}

// ProcessFile takes a file and generates an xDS snapshot
func (p *Processor) ProcessFile(file watcher.NotifyMessage) {

	// Parse file into object
	xdsControlConfig, err := parseYaml(file.FilePath)
	if err != nil {
		p.Errorf("error parsing yaml file: %+v", err)
		return
	}

	for _, service := range xdsControlConfig.Services {
		p.xdsCache.AddService(service.Name, service.Address, service.Port)
	}

	serviceResources := p.xdsCache.ServiceContents(p.nodeID)

	if len(serviceResources) == 0 {
		p.Error("Services params list empty")
		return
	}

	myResources := resources.DefaultClientResources(serviceResources)

	// TODO: Work on adding server listeners
	// inboundLis := resources.DefaultServerListener("0.0.0.0", 9101, resources.SecurityLevelNone)
	// myResources.Listeners = append(myResources.Listeners, inboundLis)

	// Create a snapshot with the passed in resources.
	resourceMap := map[v3resource.Type][]types.Resource{
		v3resource.ListenerType: resourceSlice(myResources.Listeners),
		v3resource.RouteType:    resourceSlice(myResources.Routes),
		v3resource.ClusterType:  resourceSlice(myResources.Clusters),
		v3resource.EndpointType: resourceSlice(myResources.Endpoints),
	}

	// Create the snapshot that we'll serve to Envoy
	snapshot, err := cache.NewSnapshot(
		p.newSnapshotVersion(), // version
		resourceMap,
	)

	if err := snapshot.Consistent(); err != nil {
		p.Errorf("snapshot inconsistency: %+v\n\n\n%+v", snapshot, err)
		return
	}
	p.Debugf("will serve snapshot %+v", snapshot)

	// Add the snapshot to the cache
	if err := p.cache.SetSnapshot(p.ctx, p.nodeID, snapshot); err != nil {
		p.Errorf("snapshot error %q for %+v", err, snapshot)
		os.Exit(1)
	}
}

// resourceSlice accepts a slice of any type of proto messages and returns a
// slice of types.Resource.  Will panic if there is an input type mismatch.
func resourceSlice(i interface{}) []types.Resource {
	v := reflect.ValueOf(i)
	rs := make([]types.Resource, v.Len())
	for i := 0; i < v.Len(); i++ {
		rs[i] = v.Index(i).Interface().(types.Resource)
	}
	return rs
}
