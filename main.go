package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/shirou/gopsutil/v4/cpu"
)

type DeviceType int

const (
	CPU DeviceType = iota
	HDD
	NVMe
)

func (dt DeviceType) String() string {
	switch dt {
	case CPU:
		return "cpu"
	case HDD:
		return "hdd"
	case NVMe:
		return "nvme"
	default:
		return "unknown"
	}
}

// Implement the json.Marshaler interface for DeviceType
func (dt DeviceType) MarshalJSON() ([]byte, error) {
	return json.Marshal(dt.String())
}

type TemperatureData struct {
	Type        DeviceType `json:"type"`
	ID          string     `json:"id"`
	Model       string     `json:"model,omitempty"`
	Temperature float64    `json:"temperature"`
	Zone        string     `json:"zone"`
}

func (t *TemperatureData) UpdateTemperature() error {
	var err error
	switch t.Type {
	case CPU:
		t.Temperature, err = getCPUTemperature(t.Zone)
	case HDD:
		t.Temperature, err = getHDDTemperature(t.Zone)
	case NVMe:
		t.Temperature, err = getNVMeTemperature(t.Zone)
	default:
		t.Temperature, err = 0.0, fmt.Errorf("unkown deviceType %v", t.Type)
	}
	return err
}

func NewTemperatureData(deviceType DeviceType, zone string) (*TemperatureData, error) {
	var dev *TemperatureData = nil
	switch deviceType {
	case CPU:
		zoneID := strings.Split(zone, "/")[4]
		cpuModel := "Unknown"
		cpuInfo, err := cpu.Info()
		if err == nil && len(cpuInfo) > 0 {
			cpuModel = cpuInfo[0].ModelName
		}
		dev = &TemperatureData{Type: deviceType, ID: zoneID, Model: cpuModel, Zone: zone}
	case HDD:
		output, err := getSMART(zone, []string{"-i"})
		if err != nil {
			return dev, err
		}
		hddModel := "Unkown"
		serialNumber := "Unkown"
		nfieldFound := 0
		scanner := bufio.NewScanner(strings.NewReader(output))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Serial Number") {
				serialNumber = strings.TrimSpace(strings.Split(line, ":")[1])
				nfieldFound++
			}
			if strings.Contains(line, "Device Model") {
				hddModel = strings.TrimSpace(strings.Split(line, ":")[1])
				nfieldFound++
			}
			if nfieldFound == 2 {
				break
			}
		}
		dev = &TemperatureData{Type: deviceType, ID: serialNumber, Model: hddModel, Zone: zone}
	case NVMe:
		// get the model from the nvme device using nvme list command, find the line containing the string zone, set the 4th field as the model, and the third field as the zoneID
		cmd := exec.Command("nvme", "list")
		output, err := cmd.Output()
		if err != nil {
			return dev, fmt.Errorf("failed to execute nvme list command: %v", err)
		}
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, zone) {
				fields := strings.Fields(line)
				dev = &TemperatureData{Type: deviceType, ID: fields[2], Model: fields[3], Zone: zone}
				break
			}
		}
		if dev == nil {
			return dev, fmt.Errorf("failed to find nvme device %s", zone)
		}

		dev = &TemperatureData{Type: deviceType, ID: zone, Model: "NVMe", Zone: zone}
	default:
		return dev, fmt.Errorf("unkown deviceType %v", deviceType)
	}
	err := dev.UpdateTemperature()
	return dev, err
}

func getCPUTemperature(zone string) (float64, error) {
	content, err := os.ReadFile(zone)
	if err != nil {
		return 0.0, fmt.Errorf("failed to read temperature from %s: %v", zone, err)
	}

	// Parse the temperature value (assuming millidegree Celsius)
	tempMilli, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		zoneID := strings.Split(zone, "/")[3]
		return 0.0, fmt.Errorf("failed to parse temperature for %s: %v", zoneID, err)
	}

	temp := float64(tempMilli) / 1000.0
	return temp, nil
}

func getSMART(drive string, args []string) (string, error) {
	cmd := exec.Command("smartctl", append(args, drive)...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute smartctl for %s: %v", drive, err)
	}
	return string(output), nil
}

func getHDDTemperature(drive string) (float64, error) {
	output, err := getSMART(drive, []string{"-a"})
	if err != nil {
		return 0.0, err
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	tempStr := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "194 Temp") {
			fields := strings.Fields(line)
			tempStr = fields[9]
			break
		}
		if strings.Contains(line, "190 Airflow") {
			fields := strings.Fields(line)
			tempStr = fields[9]
			break
		}
		if strings.Contains(line, "Temperature Sensor 1") {
			fields := strings.Fields(line)
			tempStr = fields[9]
			break
		}
		if strings.Contains(line, "Current Drive Temperature") {
			fields := strings.Fields(line)
			tempStr = fields[3]
			break
		}
		if strings.Contains(line, "Temperature") {
			fields := strings.Fields(line)
			tempStr = fields[3]
			break
		}
	}

	if tempStr == "" {
		return 0.0, fmt.Errorf("failed to find temperature for %s", drive)
	}

	temp, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return 0.0, fmt.Errorf("failed to parse temperature for %s: %v", drive, err)
	}
	return temp, nil
}

func getNVMeTemperature(device string) (float64, error) {
	cmd := exec.Command("nvme", "smart-log", device)
	output, err := cmd.Output()
	if err != nil {
		return 0.0, fmt.Errorf("failed to execute nvme command: %v", err)
	}

	re := regexp.MustCompile(`\d+(\.\d+)?`)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) > 1 && strings.Contains(strings.ToLower(parts[0]), "temperature") {
			match := re.FindString(parts[1])
			temp, err := strconv.ParseFloat(match, 64)
			if err == nil {
				return temp, nil
			}
		}
	}
	return 0.0, fmt.Errorf("failed to find temperature for nvme %s", device)
}

func findDevices() (int, map[DeviceType][]string, error) {
	// Find CPU zones
	cpuZones, _ := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")

	// Find HDD drives
	hddDrives, _ := filepath.Glob("/dev/sd[a-z]")

	// Find NVMe drives
	nvmeDrives, _ := filepath.Glob("/dev/nvme[0-9]n[0-9]")

	nDevices := len(cpuZones) + len(hddDrives) + len(nvmeDrives)
	if nDevices == 0 {
		return nDevices, nil, fmt.Errorf("failed to discover CPU, HDD, or NVMe devices")
	}

	// create a map of the devices, key values are cpu, hdd, nvme
	devices := make(map[DeviceType][]string)
	devices[CPU] = cpuZones
	devices[HDD] = hddDrives
	devices[NVMe] = nvmeDrives

	return nDevices, devices, nil
}

type Context struct {
	tempDB        []TemperatureData
	port          string
	cacheDuration int
	cacheMutex    sync.Mutex
	lastCacheTime int64
	endPoint      string
}

var (
	ctx  Context
	args struct {
		Daemon        bool   `arg:"-d" help:"Run as a daemon" default:"false"`
		Port          int    `arg:"-p" help:"Port number" default:"7634"`
		CacheDuration int    `arg:"--cache,-t" help:"Cache duration in seconds" default:"60"`
		EndPoint      string `arg:"--endpoint,-e" help:"Endpoint for the HTTP server" default:"/"`
		Version       bool   `arg:"-v" help:"Print version and exit"`
	}
	Version string = "dev"
)

func infoHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now().Unix()

	// Check if cache needs to be updated
	if now-ctx.lastCacheTime > int64(ctx.cacheDuration) {
		ctx.cacheMutex.Lock()
		defer ctx.cacheMutex.Unlock()

		// Double-check if another goroutine has already updated the cache
		if now-ctx.lastCacheTime > int64(ctx.cacheDuration) {
			// Update temperatures
			tempChan := make(chan struct {
				TemperatureData
				int
			}, len(ctx.tempDB))
			errChan := make(chan struct {
				error
				string
			}, len(ctx.tempDB))
			var wg sync.WaitGroup

			for i, tempData := range ctx.tempDB {
				wg.Add(1)
				go func(i int, tempData TemperatureData) {
					defer wg.Done()
					err := tempData.UpdateTemperature()
					if err != nil {
						errChan <- struct {
							error
							string
						}{err, tempData.Zone}
						return
					}
					tempChan <- struct {
						TemperatureData
						int
					}{tempData, i}
				}(i, tempData)
			}

			go func() {
				wg.Wait()
				close(tempChan)
				close(errChan)
			}()

			for tempData := range tempChan {
				ctx.tempDB[tempData.int] = tempData.TemperatureData
			}

			for err := range errChan {
				select {
				case <-r.Context().Done():
					return
				default:
					http.Error(w, fmt.Sprintf("Error updating temperature for %s: %v", err.string, err.error), http.StatusInternalServerError)
					return
				}
			}
			ctx.lastCacheTime = now
		}
	}

	// Generate JSON response
	jsonData, err := json.MarshalIndent(ctx.tempDB, "", "  ")
	if err != nil {
		http.Error(w, "Error formatting JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)

	clientEnd := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		clientEnd = xff
	}

	fmt.Printf("FROM %s: %s [cache: %s]\n", clientEnd, r.URL.Path, time.Unix(ctx.lastCacheTime, 0).Format("2006-01-02 15:04:05"))
}

// create a wrapper function of

func main() {
	// Parse command-line arguments

	// exit the program if the user is not sudoer
	if os.Geteuid() != 0 {
		fmt.Fprintf(os.Stderr, "Error: This program requires root privileges\n")
		os.Exit(1)
	}

	// exit if the program is not run on linux
	if runtime.GOOS != "linux" {
		fmt.Fprintf(os.Stderr, "Error: This program is only supported on Linux\n")
		os.Exit(1)
	}

	// check if smartctl and nvme commands are available
	if _, err := exec.LookPath("smartctl"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: smartctl command not found, HDD temperature will not be available\n")
	}
	if _, err := exec.LookPath("nvme"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: nvme command not found, NVMe temperature will not be available\n")
	}

	arg.MustParse(&args)
	ctx = Context{
		tempDB:        nil,
		cacheMutex:    sync.Mutex{},
		port:          strconv.Itoa(args.Port),
		cacheDuration: args.CacheDuration,
		lastCacheTime: 0,
		endPoint:      args.EndPoint,
	}
	daemonMode := args.Daemon
	debugWriter := os.Stdout
	if !daemonMode {
		debugWriter = os.Stderr
	}

	if args.Version {
		fmt.Printf("Thermald-go version: %s\n", Version)
		os.Exit(0)
	}

	fmt.Fprintf(debugWriter, "Daemon mode: %v\n", daemonMode)
	fmt.Fprintf(debugWriter, "Port: %s\n", ctx.port)
	fmt.Fprintf(debugWriter, "Cache duration: %d seconds\n", ctx.cacheDuration)

	// Discover CPU zones and HDD drives
	nDevices, devicesMap, err := findDevices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	tempChan := make(chan struct {
		TemperatureData
		int
	})
	errChan := make(chan struct {
		error
		string
	})

	// Create TemperatureData objects for each device
	iDevice := 0
	for _, deviceType := range []DeviceType{CPU, HDD, NVMe} {
		// if deviceType is not found in the devicesMap, skip it
		devices, ok := devicesMap[deviceType]
		if !ok {
			continue
		}
		fmt.Fprintf(debugWriter, "%s:\n", strings.ToUpper(deviceType.String()))
		for _, device := range devices {
			fmt.Fprintf(debugWriter, "  - %s\n", device)
			wg.Add(1)
			go func(deviceType DeviceType, device string, i int) {
				defer wg.Done()
				tempData, err := NewTemperatureData(deviceType, device)
				if err != nil {
					errChan <- struct {
						error
						string
					}{err, device}
					return
				}
				tempChan <- struct {
					TemperatureData
					int
				}{*tempData, i}
			}(deviceType, device, iDevice)
			iDevice++
		}
	}

	// Close channels when all goroutines are done
	go func() {
		wg.Wait()
		close(tempChan)
		close(errChan)
	}()

	// create tempDB array to retrieve the data from the tempChan
	ctx.tempDB = make([]TemperatureData, nDevices)
	for tempData := range tempChan {
		ctx.tempDB[tempData.int] = tempData.TemperatureData
	}

	// remove elements in tempDB whose temperature is 0
	for i := 0; i < len(ctx.tempDB); i++ {
		if ctx.tempDB[i].Temperature == 0 {
			ctx.tempDB = append(ctx.tempDB[:i], ctx.tempDB[i+1:]...)
			i--
		}
	}
	nValidDevices := len(ctx.tempDB)
	fmt.Fprintf(debugWriter, "Found %d devices, %d valid devices\n", nDevices, nValidDevices)

	// Handle errors
	for err := range errChan {
		fmt.Fprintf(os.Stderr, "Error for %s: %v\n", err.string, err.error)
	}

	// if not daemonMode, print the ctx as JSON and exit
	if !daemonMode {
		jsonData, err := json.MarshalIndent(ctx.tempDB, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))
		return
	}

	// // // Generate initial temperature data
	// // ctx, err = generateTemperatures(cpuZones, hddDrives)
	// // if err != nil {
	// // 	fmt.Fprintf(os.Stderr, "Error generating initial temperatures: %v\n", err)
	// // 	os.Exit(1)
	// // }

	// Daemon mode: Start HTTP server
	http.HandleFunc(ctx.endPoint, infoHandler)

	fmt.Printf("Starting server on port %s\n", ctx.port)
	if err := http.ListenAndServe(":"+ctx.port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting HTTP server: %v\n", err)
		os.Exit(1)
	}
}
