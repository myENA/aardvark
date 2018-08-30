package config

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"strings"

	template "github.com/hashicorp/go-sockaddr/template"
	log "github.com/sirupsen/logrus"
)

// Config represents application configuration
type Config struct {
	RouteID            string   // local router ID
	RouteASN           uint32   // local and upstream ASN
	RoutePeer          []string // upstream BGP peer(s)
	DockerNetwork      []string // names of watched docker network(s)
	DockerDefaultRoute net.IP   // default route within container
}

// csv2StringSlice converts comma separated values into a string slice
func csv2StringSlice(s string) []string {
	var res []string
	for _, temps := range strings.Split(s, ",") {
		res = append(res, strings.TrimSpace(temps))
	}
	return res
}

// ParseFlags parses command line arguments and populates the application configuration
func ParseFlags(args []string) (*Config, error) {
	var config *Config                // returned config
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
		config = new(Config)
	}

	// init flagset
	cmdFlags = flag.NewFlagSet("aardvark", flag.ExitOnError)

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
		return nil, err
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
		return nil, errors.New("unknown non-flag argument(s) present")
	}

	// process route ID flag
	if config.RouteID, err = template.Parse(routeIDFlag); err != nil {
		return nil, err
	}

	// cast route asn
	config.RouteASN = uint32(routeASNFlag)

	// process csv flags
	config.RoutePeer = csv2StringSlice(routePeerFlag)
	config.DockerNetwork = csv2StringSlice(dockerNetworkFlag)

	// process route ID flag
	if dockerDefaultRouteFlag != "" {
		var temps string // parsed template
		// process template
		if temps, err = template.Parse(dockerDefaultRouteFlag); err != nil {
			return nil, err
		}
		// parse resulting ip
		config.DockerDefaultRoute = net.ParseIP(temps)
		// check parse
		if config.DockerDefaultRoute == nil {
			return nil, fmt.Errorf("failed to parse default route: %s", temps)
		}
	}

	// all good
	return config, nil
}
