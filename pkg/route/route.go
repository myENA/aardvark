package route

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/myENA/aardvark/pkg/config"
	bgpConfig "github.com/osrg/gobgp/config"
	"github.com/osrg/gobgp/packet/bgp"
	gobgp "github.com/osrg/gobgp/server"
	"github.com/osrg/gobgp/table"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// package globals
var (
	routeMap  = make(map[string]containerInfo)
	routeLock sync.RWMutex
	bgpServer *gobgp.BgpServer
	appConfig *config.Config
)

// containerInfo contains container routing information
type containerInfo struct {
	Name     string
	ID       string
	Network  dc.ContainerNetwork
	pathUUID []byte
}

// Setup performs initial route configuration
func Setup(conf *config.Config) error {
	var err error // general error holder

	// copy config
	appConfig = conf

	// init BGP server
	bgpServer = gobgp.NewBgpServer()
	go bgpServer.Serve()

	// global config
	if err = bgpServer.Start(&bgpConfig.Global{
		Config: bgpConfig.GlobalConfig{
			As:       appConfig.RouteASN,
			RouterId: appConfig.RouteID,
			Port:     -1, // don't listed on tcp:179
		},
	}); err != nil {
		return err
	}

	// loop through and add configured peers
	for _, pAddr := range appConfig.RoutePeer {
		if err = bgpServer.AddNeighbor(&bgpConfig.Neighbor{
			Config: bgpConfig.NeighborConfig{
				NeighborAddress: pAddr,
			},
		}); err != nil {
			return err
		}
	}

	// all good
	return nil
}

// Add advertises bgp routes for the given container identifier
func Add(container *dc.Container) error {
	var ci containerInfo
	var nn string
	var ok, matched bool
	var err error

	// loop over configured networks
	for _, nn = range appConfig.DockerNetwork {
		if ci.Network, ok = container.NetworkSettings.Networks[nn]; ok {
			log.WithFields(log.Fields{
				"topic":             "route",
				"containerID":       container.ID,
				"containerName":     container.Name,
				"containerNetworks": container.NetworkSettings.Networks,
			}).Debugf("network matched")
			matched = true
			break
		}
	}

	// check network match
	if !matched {
		log.WithFields(log.Fields{
			"topic":             "route",
			"containerID":       container.ID,
			"containerName":     container.Name,
			"containerNetworks": container.NetworkSettings.Networks,
		}).Debugf("network not matched")
		return nil /// nothing to do here
	}

	// validate container ip info
	if ci.Network.IPAddress == "" || ci.Network.IPPrefixLen == 0 {
		log.WithFields(log.Fields{
			"topic":            "route",
			"containerID":      container.ID,
			"containerName":    container.Name,
			"containerNetwork": ci.Network,
		}).Debugf("invalid IPAddress or IPPrefixLen")
		return nil // nothing to do here
	}

	// populate misc info
	ci.ID = container.ID
	ci.Name = container.Name

	// attempt to advertise path
	if ci.pathUUID, err = bgpServer.AddPath("",
		[]*table.Path{table.NewPath(
			nil, bgp.NewIPAddrPrefix(32, ci.Network.IPAddress),
			false, []bgp.PathAttributeInterface{
				bgp.NewPathAttributeOrigin(0),
				bgp.NewPathAttributeNextHop(appConfig.RouteID),
			},
			time.Now(), false,
		)},
	); err != nil {
		return err
	}

	// update map
	routeLock.Lock()
	routeMap[container.ID] = ci
	routeLock.Unlock()

	// log add
	log.WithFields(log.Fields{
		"topic":         "route",
		"containerID":   ci.ID,
		"containerName": ci.Name,
		"containerIP":   ci.Network.IPAddress,
	}).Infof("added route")

	// replace default container route if specified
	if appConfig.DockerDefaultRoute != nil {
		if err = replaceDefaultRoute(container); err != nil {
			log.WithFields(log.Fields{
				"topic":         "event",
				"containerID":   ci.ID,
				"containerName": ci.Name,
				"containerIP":   ci.Network.IPAddress,
				"error":         err,
			}).Error("failed to replace route")
		}
	}

	// all okay
	return nil
}

// Delete removes the advertised routes for the given container identifier
func Delete(id string) error {
	var ci containerInfo
	var ok bool
	var err error

	// get read lock
	routeLock.RLock()

	// check map
	ci, ok = routeMap[id]

	// release read lock
	routeLock.RUnlock()

	if !ok {
		return nil // nothing to do
	}

	// delete path and check for error
	if err = bgpServer.DeletePath(ci.pathUUID, 0, "", nil); err != nil {
		return err
	}

	// update map
	routeLock.Lock()
	delete(routeMap, id)
	routeLock.Unlock()

	// log delete
	log.WithFields(log.Fields{
		"topic":         "route",
		"containerID":   ci.ID,
		"containerName": ci.Name,
		"containerIP":   ci.Network.IPAddress,
	}).Infof("deleted route")

	// all okay
	return nil
}

func replaceDefaultRoute(container *dc.Container) error {
	var nl *netlink.Handle                   // netlink handle
	var containerNs, originNs netns.NsHandle // netns handles
	var err error                            // error holder

	// lock os thread to prevent switching namespaces and release when done
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// get current ns
	if originNs, err = netns.Get(); err != nil {
		return err
	}

	// debugging
	log.WithFields(log.Fields{
		"topic":     "docker",
		"currentNs": originNs.String(),
	}).Debug("got current namespace")

	// get container namespace
	if containerNs, err = netns.GetFromPath(
		fmt.Sprintf("/tmp/proc/%d/ns/net",
			container.State.Pid)); err != nil {
		return err
	}

	// debugging
	log.WithFields(log.Fields{
		"topic":         "docker",
		"containerID":   container.ID,
		"containerName": container.Name,
		"containerNs":   containerNs.String(),
	}).Debug("got container namespace")

	// get netlink handle
	if nl, err = netlink.NewHandleAtFrom(containerNs, originNs); err != nil {
		return err
	}

	// attempt to replace route
	if err = nl.RouteReplace(&netlink.Route{
		Dst: nil,
		Gw:  appConfig.DockerDefaultRoute,
	}); err != nil {
		return err
	}

	// log update
	log.WithFields(log.Fields{
		"topic":         "docker",
		"containerID":   container.ID,
		"containerName": container.Name,
		"defaultRoute":  appConfig.DockerDefaultRoute.String(),
	}).Infof("updated default route")

	// all good
	return nil
}
