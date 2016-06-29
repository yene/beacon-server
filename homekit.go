// +build ignore

package homekitexample

import (
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"

	"github.com/BurntSushi/toml"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
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

var config Config

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

	beaconUUID, _ := gatt.ParseUUID(config.BeaconUUID)
	beaconS := fmt.Sprintf("%x", a.ManufacturerData[4:20])

	if beaconUUID.String() != beaconS {
		return
	}

	config.lastseen = time.Now()
	config.didAlarm = false
	acc.Outlet.OutletInUse.SetValue(true)

	fmt.Println("Found beacon I was searching for:")
	fmt.Println("  uuid", beaconS)
	fmt.Println("  major", binary.BigEndian.Uint16(a.ManufacturerData[20:22]))
	fmt.Println("  minor", binary.BigEndian.Uint16(a.ManufacturerData[22:24]))
	power := int8(a.ManufacturerData[24])
	fmt.Println("  power", power)
	fmt.Println("--------------------")
}

var acc *accessory.Outlet

func main() {
	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		fmt.Println(err)
		return
	}

	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered))
	d.Init(onStateChanged)

	go reportMissing()

	info := accessory.Info{
		Name: "iBeacon",
	}
	acc = accessory.NewOutlet(info)

	acc.Outlet.OutletInUse.SetValue(false)

	config := hc.Config{Pin: "00102003"}
	t, err := hc.NewIPTransport(config, acc.Accessory)
	if err != nil {
		log.Fatal(err)
	}

	hc.OnTermination(func() {
		t.Stop()
	})

	t.Start()
}

func reportMissing() {
	for {
		if !config.didAlarm && time.Since(config.lastseen) > time.Second*5 {
			fmt.Println("iBeacon went missing, alarm the cops")
			config.didAlarm = true
			acc.Outlet.OutletInUse.SetValue(false)
		}
		time.Sleep(time.Second * 1)
	}
}

func isBeacon(m []byte) bool {
	var id uint16 = 0x004C
	return len(m) == 25 && m[0] == uint8(id) && m[1] == uint8(id>>8) && m[2] == 0x02 && m[3] == 0x15
}
