package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/yene/gatt"
	"github.com/yene/gatt/examples/option"
)

type Rules struct {
	BeaconUUID   string `json:"uuid"`
	BeaconMajor  string `json:"major"`
	BeaconMinor  string `json:"minor"`
	WebhookEnter string `json:"enter"`
	WebhookLeave string `json:"leave"`
}

type Beacon struct {
	UUID     string    `json:"uuid"`
	Major    int       `json:"major"`
	Minor    int       `json:"minor"`
	Power    int8      `json:"power"`
	LastSeen time.Time `json:"-"`
}

const httpAddr = ":8080"
const heartBeat = time.Second * 5 // Apple advertising interval is 100ms, others use up to 1000ms
const didEnter = true
const didLeave = false

var rules []Rules

var foundBeacons []Beacon

func main() {
	foundBeacons = make([]Beacon, 0)
	loadRules()

	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered))
	d.Init(onStateChanged)

	go checkForMissingBeacon()

	http.Handle("/", http.FileServer(assetFS()))

	http.HandleFunc("/rules.json", func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "POST" {
			decoder := json.NewDecoder(r.Body)
			err := decoder.Decode(&rules)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeRules()
		} else {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(rules); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	})

	http.HandleFunc("/list.json", func(w http.ResponseWriter, r *http.Request) {
		//v = append(v, Beacon{"e2c56db5dffb48d2b060d0f5a71096e0", 1, 0, -50, time.Now()})
		//w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(foundBeacons); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	fmt.Println("listen on", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, nil))

}

func onStateChanged(d gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("scanning...")
		d.Scan([]gatt.UUID{}, true)
		return
	default:
		d.StopScanning()
	}
}

func onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	if !isBeacon(a.ManufacturerData) {
		return
	}

	beacon := parseBeacon(a.ManufacturerData)
	if beaconExists(beacon.UUID, beacon.Major, beacon.Minor) {
		updateBeacon(beacon.UUID, beacon.Major, beacon.Minor)
	} else {
		foundBeacons = append(foundBeacons, beacon)
		runRulesFor(beacon, didEnter)
		fmt.Println("Beacon", beacon.UUID, beacon.Major, beacon.Minor, "did enter")
	}
}

func checkForMissingBeacon() {
	for {
		for i := len(foundBeacons) - 1; i >= 0; i-- {
			b := foundBeacons[i]
			if time.Since(b.LastSeen) > heartBeat {
				runRulesFor(b, didLeave)
				fmt.Println("Beacon", b.UUID, b.Major, b.Minor, "did leave")
				foundBeacons = append(foundBeacons[:i], foundBeacons[i+1:]...)
			}
		}
		time.Sleep(time.Second * 1)
	}
}

func isBeacon(m []byte) bool {
	var id uint16 = 0x004C
	return len(m) == 25 && m[0] == uint8(id) && m[1] == uint8(id>>8) && m[2] == 0x02 && m[3] == 0x15
}

func parseBeacon(m []byte) Beacon {
	return Beacon{
		UUID:     fmt.Sprintf("%x", m[4:20]),
		Major:    int(binary.BigEndian.Uint16(m[20:22])),
		Minor:    int(binary.BigEndian.Uint16(m[22:24])),
		Power:    int8(m[24]),
		LastSeen: time.Now(),
	}
}

func loadRules() {
	file, err := ioutil.ReadFile("rules.json")

	if err != nil {
		rules = make([]Rules, 0)
		fmt.Println("No \"rules.json\" found, creating empty.")
		writeRules()
		return
	}

	if err := json.Unmarshal(file, &rules); err != nil {
		panic(err)
	}
}

func writeRules() {
	data, err := json.Marshal(rules)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("rules.json", data, 0644)
	if err != nil {
		panic(err)
	}
}

func requestURL(url string) {
	response, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	if response.StatusCode != http.StatusOK {
		fmt.Println("response Status:", response.Status)
	}
}

func beaconExists(uuid string, major int, minor int) bool {
	for _, b := range foundBeacons {
		if b.UUID == uuid && b.Major == major && b.Minor == minor {
			return true
		}
	}
	return false
}

func updateBeacon(uuid string, major int, minor int) {
	for i, b := range foundBeacons {
		if b.UUID == uuid && b.Major == major && b.Minor == minor {
			ub := b
			ub.LastSeen = time.Now()
			foundBeacons[i] = ub
		}
	}
}

func runRulesFor(b Beacon, enter bool) {
	for _, r := range rules {
		if r.BeaconUUID != b.UUID {
			continue
		}

		if r.BeaconMajor != "*" {
			if r.BeaconMajor != strconv.Itoa(b.Major) {
				continue
			}
		}

		if r.BeaconMinor != "*" {
			if r.BeaconMinor != strconv.Itoa(b.Minor) {
				continue
			}
		}

		if enter {
			requestURL(r.WebhookEnter)
		} else {
			requestURL(r.WebhookLeave)
		}

	}
}
