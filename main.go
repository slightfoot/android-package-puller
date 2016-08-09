// 
// Android Package Puller
// 
// Simple command to pull android packages from connected devices.
// 
// This work is licensed under a Creative Commons Attribution-ShareAlike 4.0 International License.
// http://creativecommons.org/licenses/by-sa/4.0/
// 

package main // github.com/slightfoot/android-package-puller

import (
	"android.googlesource.com/platform/tools/gpu/adb"
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type (
	Package struct {
		Name string
		Path string
	}
)

var (
	packageName  string
	deviceSerial string
)

func init() {
	var defaultPackageName string = ""
	var defaultDeviceSerial string = ""
	if len(os.Args) > 1 {
		defaultPackageName = os.Args[1]
	}
	if len(os.Args) > 2 {
		defaultDeviceSerial = os.Args[2]
	}

	flag.StringVar(&packageName, "package", defaultPackageName,
		"Application packageName you'd like to pull from device.")

	flag.StringVar(&deviceSerial, "device", defaultDeviceSerial,
		"Device serial number used to identify specific device.")

	flag.CommandLine.Usage = func() {
		fmt.Fprintf(os.Stderr, "Android Package Puller - " +
			"Simon Lightfoot github.com/slightfoot\n" + 
			"Usage: %s [package] [device]\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {

	// Handle command line flags
	flag.Parse()

	// Get list of devices
	devices, err := adb.Devices()
	if err != nil {
		printError("Failed to get list of devices: %s\n", err.Error())
		os.Exit(1)
	}

	// Choose device if multiple devices attached to machine
	device, err := getDevice(devices)
	if err != nil {
		os.Exit(1)
	}

	// Get list of packages from device
	pkgs, err := getPackageList(device)
	if err != nil {
		os.Exit(1)
	}

	// Choose package
	pkg, err := getPackage(pkgs)
	if err != nil {
		os.Exit(1)
	}

	// Pull package from device
	apkName := pkg.Name + ".apk"
	fmt.Printf("Pulling %s from device... ", apkName)
	err = device.Pull(pkg.Path, apkName)
	if err == nil {
		fmt.Printf("Success\n")
	} else {
		fmt.Printf("Failed\n")
		printError("Failed to pull package from device: %s", err.Error())
		os.Exit(1)
	}
}

func getDevice(devices []*adb.Device) (*adb.Device, error) {
	// Did the user ask for a specific device?if located, return it
	if len(deviceSerial) > 0 {
		for _, device := range devices {
			if device.Serial == deviceSerial {
				return device, nil
			}
		}
		fmt.Printf("Warning: Could not locate device.\n")
	}

	// Only one device? lets use it
	if len(devices) == 1 {
		fmt.Printf("Device: %s\n", devices[0].Serial)
		return devices[0], nil
	}

	// Otherwise we show a list of devices to the user
	fmt.Printf("Devices:\n")
	for i, device := range devices {
		fmt.Printf("\t%d:\t%s %s\n", i, device.Serial, device.State)
	}
	index, err := readInputNumber("Which device?", 0, len(devices)-1)
	if err != nil {
		return nil, err
	}

	return devices[index], nil
}

func getPackageList(device *adb.Device) ([]*Package, error) {
	cmd := device.Command("pm", "list", "packages", "-f")
	data, err := cmd.Call()
	if err != nil {
		fmt.Printf("Failed list root: %s\n", err)
		return nil, printError("Failed to retrieve packages: %s", err.Error())
	}

	lines := strings.Split(data, "\n")
	packages := make([]*Package, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package:") == false {
			continue
		}
		parts := strings.SplitN(line[8:], "=", 2)
		if len(parts) != 2 {
			printError("Bad package manager response: '%s'", line)
			continue
		}
		packages = append(packages, &Package{Path: parts[0], Name: parts[1]})
	}

	if len(packages) == 0 {
		return nil, printError("No packages found")
	}

	return packages, nil
}

func getPackage(pkgs []*Package) (*Package, error) {
	// Did the user ask for a specific package? if located, return it
	if len(packageName) > 0 {
		for _, pkg := range pkgs {
			if pkg.Name == packageName {
				return pkg, nil
			}
		}
		return nil, printError("Could not locate package: %s", packageName)
	}

	fmt.Printf("Packages:\n")
	for i, pkg := range pkgs {
		fmt.Printf("\t%d:\t%s %s\n", i, pkg.Name, pkg.Path)
	}
	index, err := readInputNumber("Which package?", 0, len(pkgs)-1)
	if err != nil {
		return nil, err
	}

	return pkgs[index], nil
}

func readInputNumber(prompt string, min int, max int) (int, error) {
	fmt.Printf("%s [%d-%d]: ", prompt, min, max)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err == nil {
		input = strings.TrimSpace(input)
	}
	if err != nil || len(input) == 0 {
		return -1, printError("Input cancelled")
	}

	index, err := strconv.Atoi(input)
	if err != nil {
		return -1, printError("Invalid input: %s", err.Error())
	}
	if index < min || index > max {
		return -1, printError("Input out of range: %d", index)
	}

	return index, nil
}

func printError(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	return err
}
