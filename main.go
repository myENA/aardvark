package main

import (
	"os"

	"github.com/myENA/aardvark/pkg/config"
	"github.com/myENA/aardvark/pkg/docker"
	"github.com/myENA/aardvark/pkg/route"
	log "github.com/sirupsen/logrus"
)

// appMain performs application initialization and returns and exit code
func appMain() int {
	var appConfig *config.Config // application config
	var err error                // general error holder

	// set log level and formatter
	log.SetLevel(log.DebugLevel)

	// process flags
	if appConfig, err = config.ParseFlags(os.Args[1:]); err != nil {
		log.WithFields(log.Fields{
			"component": "setup",
			"error":     err,
		}).Error("flag setup failed")
		return 1
	}

	// setup docker client
	if err = docker.Setup(); err != nil {
		log.WithFields(log.Fields{
			"component": "setup",
			"error":     err,
		}).Error("initial docker setup failed")
		return 1
	}

	// setup route engine
	if err = route.Setup(appConfig); err != nil {
		log.WithFields(log.Fields{
			"component": "setup",
			"error":     err,
		}).Error("initial route setup failed")
		return 1
	}

	// perform initial route sync
	if err = docker.Sync(); err != nil {
		log.WithFields(log.Fields{
			"topic": "setup",
			"error": err,
		}).Error("initial sync failed")
		return 1
	}

	// spawn event handler and wait ...
	if err = docker.Handler(); err != nil {
		log.WithFields(log.Fields{
			"topic": "docker",
			"error": err,
		}).Error("event handler failed")
		// all done
		return 1
	}

	// exit clean
	return 0
}

// the main function that exits on completion
func main() {
	os.Exit(appMain())
}
