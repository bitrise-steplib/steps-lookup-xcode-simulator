package main

import (
	"fmt"
	"github.com/bitrise-steplib/bitrise-step-look-up-xcode-simulator-udid/destination"
	"os"
	"strings"
	"time"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
	"github.com/bitrise-io/go-xcode/simulator"
)

// Input ...
type Input struct {
	Destination string `env:"destination,required"`
}

func run() error {
	var input Input
	if err := stepconf.Parse(&input); err != nil {
		return fmt.Errorf("issue with input: %s", err)
	}

	stepconf.Print(input)
	fmt.Println()

	var sim simulator.InfoModel
	var osVersion string

	simulatorDestination, err := destination.NewSimulator(input.Destination)
	if err != nil {
		return fmt.Errorf("invalid destination specifier: %v", err)
	}

	platform := strings.TrimSuffix(simulatorDestination.Platform, " Simulator")
	// Retry gathering device information since xcrun simctl list can fail to show the complete device list
	if err := retry.Times(3).Wait(10 * time.Second).Try(func(attempt uint) error {
		var errGetSimulator error
		if simulatorDestination.OS == "latest" {
			simulatorDevice := simulatorDestination.Name
			if simulatorDevice == "iPad" {
				log.Warnf("Given device (%s) is deprecated, using iPad Air (3rd generation)...", simulatorDevice)
				simulatorDevice = "iPad Air (3rd generation)"
			}

			sim, osVersion, errGetSimulator = simulator.GetLatestSimulatorInfoAndVersion(platform, simulatorDevice)
		} else {
			normalizedOsVersion := simulatorDestination.OS
			osVersionSplit := strings.Split(normalizedOsVersion, ".")
			if len(osVersionSplit) > 2 {
				normalizedOsVersion = strings.Join(osVersionSplit[0:2], ".")
			}
			osVersion = fmt.Sprintf("%s %s", platform, normalizedOsVersion)

			sim, errGetSimulator = simulator.GetSimulatorInfo(osVersion, simulatorDestination.Name)
		}

		if errGetSimulator != nil {
			log.Warnf("attempt %d to get simulator UDID failed with error: %s", attempt, errGetSimulator)
		}

		return errGetSimulator
	}); err != nil {
		return fmt.Errorf("simulator UDID lookup failed: %s", err)
	}

	log.Infof("Found Simulator (UDID: %s OS version: %s", sim.ID, osVersion)

	const simulatorUDIDKey = "XCODE_SIMULATOR_UDID"
	log.Infof("Exporting %s -> %s", simulatorUDIDKey, sim.ID)
	if err := tools.ExportEnvironmentWithEnvman("XCODE_SIMULATOR_UDID", sim.ID); err != nil {
		return fmt.Errorf("Failed to export Simulator UDID: %v", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Errorf("Error: %s", err)
		os.Exit(1)
	}
}
