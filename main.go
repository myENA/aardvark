package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

// appMain performs application initialization and returns and exit code
func appMain() int {
	var err error // general error holder

	// set log level and formatter
	log.SetLevel(log.DebugLevel)

	// process flags
	if err = parseFlags(os.Args[1:]); err != nil {
		log.WithFields(log.Fields{
			"component": "setup",
			"error":     err,
		}).Error("flag setup failed")
		return 1
	}

	// setup docker client
	if err = dockerSetup(); err != nil {
		log.WithFields(log.Fields{
			"component": "setup",
			"error":     err,
		}).Error("initial docker setup failed")
		return 1
	}

	// setup route engine
	if err = routeSetup(); err != nil {
		log.WithFields(log.Fields{
			"component": "setup",
			"error":     err,
		}).Error("initial route setup failed")
		return 1
	}

	// perform initial route sync
	if err = routeSync(); err != nil {
		log.WithFields(log.Fields{
			"topic": "setup",
			"error": err,
		}).Error("initial sync failed")
		return 1
	}

	// start listener
	if err = dockerListen(); err != nil {
		log.WithFields(log.Fields{
			"topic": "docker",
			"error": err,
		}).Error("failed to start event listener")
	}

	// spawn event handler and wait ...
	return dockerEventHandler()
}

// the main function that exits on completion
func main() {
	os.Exit(appMain())
}
