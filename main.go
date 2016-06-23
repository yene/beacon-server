package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/currantlabs/gatt"
	"github.com/currantlabs/gatt/examples/option"

	"github.com/BurntSushi/toml"
)

type Config struct {
	BeaconUUID  string
	BeaconMajor int
	BeaconMinor int
	BeaconMAC   string
	Webhook     string
	lastseen    time.Time
	didAlarm    bool
	HeartBeat   int
}

type Beacon struct {
	UUID  string `json:"uuid"`
	Major int    `json:"major"`
	Minor int    `json:"minor"`
	Power int    `json:"power"`
}

const httpAddr = ":8080"

var config Config

var foundBeacons map[string]Beacon

func main() {
	foundBeacons = make(map[string]Beacon)

	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		fmt.Println(err)
		return
	}

	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered))
	d.Init(onStateChanged)

	go reportMissing()

	http.HandleFunc("/list.json", func(w http.ResponseWriter, r *http.Request) {
		v := make([]Beacon, 0, len(foundBeacons))
		for _, value := range foundBeacons {
			v = append(v, value)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(v); err != nil {
			panic(err)
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
	foundBeacons[beacon.UUID] = beacon

	fmt.Println("found beacon", beacon.UUID)

}

func reportMissing() {
	for {
		if !config.didAlarm && time.Since(config.lastseen) > time.Second*5 {
			fmt.Println("iBeacon went missing, alarm the cops")
			config.didAlarm = true
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
		UUID:  fmt.Sprintf("%x", m[4:20]),
		Major: int(binary.BigEndian.Uint16(m[20:22])),
		Minor: int(binary.BigEndian.Uint16(m[22:24])),
		Power: int(m[24]),
	}
}
