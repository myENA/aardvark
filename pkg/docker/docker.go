package docker

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/myENA/aardvark/pkg/route"
	log "github.com/sirupsen/logrus"
)

// package globals
var (
	events    chan *dc.APIEvents
	client    *dc.Client
	listening bool
)

// Client returns the package docker client
func Client() *dc.Client {
	return client
}

// Setup initializes the docker connection
func Setup() error {
	var err error // error holder

	// export env if not present
	if os.Getenv("DOCKER_HOST") == "" {
		if runtime.GOOS != "windows" {
			if err = os.Setenv("DOCKER_HOST", "unix:///tmp/docker.sock"); err != nil {
				return err
			}
		} else {
			if err = os.Setenv("DOCKER_HOST", "npipe:////./pipe/docker_engine"); err != nil {
				return err
			}
		}
	}

	// initialize the event channel
	events = make(chan *dc.APIEvents)

	// initialize docker client
	client, err = dc.NewClientFromEnv()

	// all done
	return err
}

// Sync loops through existing containers for initial sync
func Sync() error {
	var apiContainers []dc.APIContainers
	var apiContainer dc.APIContainers
	var container *dc.Container
	var err error

	// get all apiContainers
	if apiContainers, err = client.ListContainers(dc.ListContainersOptions{}); err != nil {
		return err
	}

	// debugging
	log.WithFields(log.Fields{
		"topic":        "route",
		"apiContainer": apiContainer,
	}).Debug("route sync")

	// loop through apiContainers
	for _, apiContainer = range apiContainers {
		// inspect container and check error
		if container, err = client.InspectContainer(apiContainer.ID); err != nil {
			log.WithFields(log.Fields{
				"topic":       "route",
				"containerID": apiContainer.ID,
				"error":       err,
			}).Error("docker inspect failed")
			return err
		}
		// attempt to add route
		if err = route.Add(container); err != nil {
			log.WithFields(log.Fields{
				"topic":         "route",
				"containerID":   container.ID,
				"containerName": container.Name,
				"error":         err,
			}).Error("failed to sync")
		}
	}

	// all okay
	return nil
}

// Listen is a helper function to start an event listener
func listen() error {
	if !listening {
		// start listener and check for error
		if err := client.AddEventListener(events); err != nil {
			return err
		}
		// toggle listening
		listening = true
	}
	// all good
	return nil
}

// Hangup is a helper function to remove an event listener
func hangup() error {
	if listening {
		if err := client.RemoveEventListener(events); err != nil {
			return err
		}
		// toggle listening
		listening = false
	}
	// all good
	return nil
}

// Handler enters an infinite loop listening for and processing events
func Handler() error {
	var sigChan = make(chan os.Signal, 1) // signal channel
	var sig os.Signal                     // trapped signal
	var event *dc.APIEvents               // captured event
	var container *dc.Container           // container object
	var err error                         // error holder

	// trap signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// start listener
	if err = listen(); err != nil {
		return err
	}

	// loop till done
	for {
		select {
		// catch events
		case event = <-events:
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
					// inspect container and check error
					if container, err = client.InspectContainer(event.Actor.ID); err != nil {
						log.WithFields(log.Fields{
							"topic":       "route",
							"containerID": event.Actor.ID,
							"error":       err,
						}).Error("docker inspect failed")
						return err
					}
					// attempt to add route
					if err = route.Add(container); err != nil {
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
					if err = route.Delete(event.Actor.ID); err != nil {
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
		case sig = <-sigChan:
			// log signal, stop listener and break loop
			log.WithFields(log.Fields{
				"topic":  "event",
				"signal": sig,
			}).Info("stopping listener on signal")
			// hangup and return
			return hangup()
		}
	}
}
