package main

import (
	"fmt"
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
	// Simulator Configs
	SimulatorPlatform  string `env:"simulator_platform,required"`
	SimulatorDevice    string `env:"simulator_device,required"`
	SimulatorOsVersion string `env:"simulator_os_version,required"`
}

func run() error {
	var input Input
	if err := stepconf.Parse(&input); err != nil {
		return fmt.Errorf("issue with input: %s", err)
	}

	stepconf.Print(input)
	fmt.Println()

	// validate simulator related inputs
	var sim simulator.InfoModel
	var osVersion string

	platform := strings.TrimSuffix(input.SimulatorPlatform, " Simulator")
	// Retry gathering device information since xcrun simctl list can fail to show the complete device list
	if err := retry.Times(3).Wait(10 * time.Second).Try(func(attempt uint) error {
		var errGetSimulator error
		if input.SimulatorOsVersion == "latest" {
			sim, osVersion, errGetSimulator = simulator.GetLatestSimulatorInfoAndVersion(platform, input.SimulatorDevice)
		} else {
			normalizedOsVersion := input.SimulatorOsVersion
			osVersionSplit := strings.Split(normalizedOsVersion, ".")
			if len(osVersionSplit) > 2 {
				normalizedOsVersion = strings.Join(osVersionSplit[0:2], ".")
			}
			osVersion = fmt.Sprintf("%s %s", platform, normalizedOsVersion)

			sim, errGetSimulator = simulator.GetSimulatorInfo(osVersion, input.SimulatorDevice)
		}

		if errGetSimulator != nil {
			log.Warnf("attempt %d to get simulator udid failed with error: %s", attempt, errGetSimulator)
		}

		return errGetSimulator
	}); err != nil {
		return fmt.Errorf("simulator UDID lookup failed: %s", err)
	}

	log.Infof("Found Simulator (UDUD: %s OS version: %s", sim.ID, osVersion)

	const simulatorUDIDKey = "XCODE_SIMULATOR_UDID"
	log.Infof("Exporting %s -> %s", simulatorUDIDKey, sim.ID)
	if err := tools.ExportEnvironmentWithEnvman("XCODE_SIMULATOR_UDID", sim.ID); err != nil {
		return fmt.Errorf("Failed to export Simulator UDID: %v", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Errorf("%s", err)
		os.Exit(1)
	}
}
