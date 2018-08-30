package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"strings"

	template "github.com/hashicorp/go-sockaddr/template"
	log "github.com/sirupsen/logrus"
)

// appConfig represents application configuration
type appConfig struct {
	routeID            string   // local router ID
	routeASN           uint32   // local and upstream ASN
	routePeer          []string // upstream BGP peer(s)
	dockerNetwork      []string // names of watched docker network(s)
	dockerDefaultRoute net.IP   // default route within container
}

// config is a global instance of the application configuration
var config *appConfig

// csv2StringSlice converts comma separated values into a string slice
func csv2StringSlice(s string) []string {
	var res []string
	for _, temps := range strings.Split(s, ",") {
		res = append(res, strings.TrimSpace(temps))
	}
	return res
}

// parseFlags parses command line arguments and populates the application configuration
func parseFlags(args []string) error {
	var cmdFlags *flag.FlagSet        // app flagset
	var routeIDFlag string            // raw un-processed router ID or go-sockaddr template
	var routeASNFlag uint             // un-cast input
	var routePeerFlag string          // csv form of route peers
	var dockerNetworkFlag string      // csv form of docker network(s)
	var dockerDefaultRouteFlag string // raw un-processed default container route or go-sockaddr template
	var logPlainFlag bool             // toggle plain logging
	var err error                     // error holder

	// init config if needed
	if config == nil {
		config = new(appConfig)
	}

	// init flagset
	cmdFlags = flag.NewFlagSet(appName, flag.ExitOnError)

	// declare flags
	cmdFlags.BoolVar(&logPlainFlag, "text", false,
		"enable plain-text logging - json if not specified")
	cmdFlags.StringVar(&routeIDFlag, "id", "{{ GetPrivateIP }}",
		"local router ID and next-hop or go-sockaddr template")
	cmdFlags.UintVar(&routeASNFlag, "asn", 65123,
		"local and remote peer ASN")
	cmdFlags.StringVar(&routePeerFlag, "peer", "",
		"upstream BGP peer(s) in CSV format")
	cmdFlags.StringVar(&dockerNetworkFlag, "network", "weave",
		"watched Docker network(s) in CSV format")
	cmdFlags.StringVar(&dockerDefaultRouteFlag, "defaultRoute", "",
		"container default route or go-sockaddr template")

	// parse flags
	if err = cmdFlags.Parse(args); err != nil {
		return err
	}

	// set log format
	if logPlainFlag {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
	} else {
		log.SetFormatter(&log.JSONFormatter{})
	}

	// check for remaining garbage
	if cmdFlags.NArg() > 0 {
		return errors.New("unknown non-flag argument(s) present")
	}

	// process route ID flag
	if config.routeID, err = template.Parse(routeIDFlag); err != nil {
		return err
	}

	// cast route asn
	config.routeASN = uint32(routeASNFlag)

	// process csv flags
	config.routePeer = csv2StringSlice(routePeerFlag)
	config.dockerNetwork = csv2StringSlice(dockerNetworkFlag)

	// process route ID flag
	if dockerDefaultRouteFlag != "" {
		var temps string // parsed template
		// process template
		if temps, err = template.Parse(dockerDefaultRouteFlag); err != nil {
			return err
		}
		// parse resulting ip
		config.dockerDefaultRoute = net.ParseIP(temps)
		// check parse
		if config.dockerDefaultRoute == nil {
			return fmt.Errorf("failed to parse default route: %s", temps)
		}
	}

	// all good
	return nil
}
