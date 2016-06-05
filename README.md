# beacon-discover

Check if the given iBeacon is in range. Fake a HomeKit outlet that is plugged in when the iBeacon is in reach. Allows you to setup rules like: "When iBeacon discovered unlock the door."

Program needs to be run as root, tested on Raspberry PI 3.

![hardware](hardware.jpg)

![beacon found](found.png)
![beacon not found](notfound.png)

## TODO
- [ ] why does it not take my UUID?
- [ ] how to check minor and major
- [ ] heartbeat for iBeacon
- [X] interface to HomeKit

## Notes
* https://github.com/mlwelles/BeaconScanner#how-it-works
* [Gatt](https://github.com/paypal/gatt)
