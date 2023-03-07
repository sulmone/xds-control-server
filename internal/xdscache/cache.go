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

package xdscache

import (
	v3clusterpb "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	v3endpointpb "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	v3listenerpb "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	v3routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/stevesloka/envoy-xds-server/internal/resources"
)

type XDSCache struct {
	ServerListeners map[string]resources.ServerListener
	Listeners       map[string]resources.Listener
	Routes          map[string]resources.Route
	Clusters        map[string]resources.Cluster
	Endpoints       map[string]resources.Endpoint
}

func (xds *XDSCache) ClusterContents() []*v3clusterpb.Cluster {
	var r []*v3clusterpb.Cluster

	for _, c := range xds.Clusters {
		r = append(r, resources.MakeCluster(c.Name))
	}

	return r
}

func (xds *XDSCache) RouteContents() []*v3routepb.RouteConfiguration {
	var routesArray []resources.Route
	for _, r := range xds.Routes {
		routesArray = append(routesArray, r)
	}
	return []*v3routepb.RouteConfiguration{resources.MakeRoute(routesArray)}
}

func (xds *XDSCache) ListenerContents() []*v3listenerpb.Listener {
	var r []*v3listenerpb.Listener

	for _, l := range xds.Listeners {
		r = append(r, resources.MakeHTTPListener(l.Name, l.RouteNames[0], l.Address, l.Port))
	}

	for _, sl := range xds.ServerListeners {
		r = append(r, resources.DefaultServerListener(sl.Address, sl.Port, resources.SecurityLevelNone))
	}
	return r
}

func (xds *XDSCache) EndpointsContents() []*v3endpointpb.ClusterLoadAssignment {
	var r []*v3endpointpb.ClusterLoadAssignment

	for _, c := range xds.Clusters {
		r = append(r, resources.MakeEndpoint(c.Name, c.Endpoints))
	}

	return r
}

func (xds *XDSCache) AddServerListener(name string, address string, port uint32) {
	xds.ServerListeners[name] = resources.ServerListener{
		Name:    name,
		Address: address,
		Port:    port,
	}
}

func (xds *XDSCache) AddListener(name string, routeNames []string, address string, port uint32) {
	xds.Listeners[name] = resources.Listener{
		Name:       name,
		Address:    address,
		Port:       port,
		RouteNames: routeNames,
	}
}

func (xds *XDSCache) AddRoute(name, prefix string, clusters []string) {
	xds.Routes[name] = resources.Route{
		Name:    name,
		Prefix:  prefix,
		Cluster: clusters[0],
	}
}

func (xds *XDSCache) AddCluster(name string) {
	xds.Clusters[name] = resources.Cluster{
		Name: name,
	}
}

func (xds *XDSCache) AddEndpoint(clusterName, upstreamHost string, upstreamPort uint32) {
	cluster := xds.Clusters[clusterName]

	cluster.Endpoints = append(cluster.Endpoints, resources.Endpoint{
		UpstreamHost: upstreamHost,
		UpstreamPort: upstreamPort,
	})

	xds.Clusters[clusterName] = cluster
}
