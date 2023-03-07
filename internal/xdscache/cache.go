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
	"fmt"
	"net"

	"github.com/sulmone/xds-control-server/internal/resources"
)

type XDSCache struct {
	Services map[string]resources.Service
}

func (xds *XDSCache) AddService(name string, address string, port uint32) {
	xds.Services[name] = resources.Service{
		Name:    name,
		Address: address,
		Port:    port,
	}
}

func (xds *XDSCache) ServiceContents(nodeID string) []*resources.ResourceParams {
	var r []*resources.ResourceParams

	for _, service := range xds.Services {
		print(service.Address)
		ips, err := net.LookupIP(service.Address)
		if err != nil {
			fmt.Printf("Could not get IPs for %s: %v\n", service.Address, err)
		}
		ip := ips[0].String()
		params := resources.ResourceParams{
			DialTarget: service.Name,
			NodeID:     nodeID,
			Host:       ip,
			Port:       service.Port,
			SecLevel:   resources.SecurityLevelNone,
		}

		r = append(r, &params)
	}
	return r
}
