package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	dc "github.com/fsouza/go-dockerclient"
	log "github.com/sirupsen/logrus"
)

// package globals
var (
	dockerEvents    chan *dc.APIEvents
	dockerClient    *dc.Client
	dockerListening bool
)

// dockerSetup initializes the docker connection
func dockerSetup() error {
	var err error // error holder

	// export env if not present
	if os.Getenv("DOCKER_HOST") == "" {
		if runtime.GOOS != "windows" {
			os.Setenv("DOCKER_HOST", "unix:///tmp/docker.sock")
		} else {
			os.Setenv("DOCKER_HOST", "npipe:////./pipe/docker_engine")
		}
	}

	// initialize the event channel
	dockerEvents = make(chan *dc.APIEvents)

	// initialize docker client
	if dockerClient, err = dc.NewClientFromEnv(); err != nil {
		return err
	}

	// all good
	return nil
}

// dockerListen is a helper function to start an event listener
func dockerListen() error {
	if !dockerListening {
		// start listener and check for error
		if err := dockerClient.AddEventListener(dockerEvents); err != nil {
			return err
		}
		// toggle listening
		dockerListening = true
	}
	// all good
	return nil
}

// dockerHangup is a helper function to remove an event listener
func dockerHangup() error {
	if dockerListening {
		if err := dockerClient.RemoveEventListener(dockerEvents); err != nil {
			return err
		}
		// toggle listening
		dockerListening = false
	}
	// all good
	return nil
}

// dockerEventHandler enters an infinite loop listening for and processing events
func dockerEventHandler() int {
	// trap signals
	var sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	// loop till done
eventLoop:
	for {
		select {
		// catch events
		case event := <-dockerEvents:
			if event != nil {
				var name string // scoped var for container name
				var ok bool     // scoped check var
				// attempt to get container name
				if name, ok = event.Actor.Attributes["name"]; !ok {
					name = "unknown"
				}
				// switch on even status
				switch event.Status {
				case "start":
					// log start event
					log.WithFields(log.Fields{
						"topic":     "event",
						"actorID":   event.Actor.ID,
						"actorName": name,
					}).Debug("container started")
					// attempt to add route
					if err := routeAdd(event.Actor.ID); err != nil {
						log.WithFields(log.Fields{
							"topic":         "route",
							"containerID":   event.Actor.ID,
							"containerName": name,
							"error":         err,
						}).Error("add failed")
					}
				case "die":
					// log die event
					log.WithFields(log.Fields{
						"topic":         "event",
						"containerID":   event.Actor.ID,
						"containerName": name,
					}).Debug("container stopped")
					// attempt to remove route
					if err := routeDelete(event.Actor.ID); err != nil {
						log.WithFields(log.Fields{
							"topic":         "route",
							"containerID":   event.Actor.ID,
							"containerName": name,
							"error":         err,
						}).Error("delete failed")
					}
				}
			}
			// watch for signals
		case sig := <-sigChan:
			// log signal, stop listener and break loop
			log.WithFields(log.Fields{
				"topic":  "event",
				"signal": sig,
			}).Info("stopping listener on signal")
			dockerHangup()
			break eventLoop
		}
	}

	// return clean
	return 0
}
