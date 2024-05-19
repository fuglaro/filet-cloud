# ⛅ Filet Cloud: Energy Efficient High Capacity Deployment on Raspberry Pi Zero 2 W

This deployment aims to provide a meaty storage capacity and speedy response, with a very low energy utilisation.

## Hardware
* Raspberry Pi Zero 2 W https://www.raspberrypi.com/products/raspberry-pi-zero-2-w/
* 32GB SanDisk Ultra MicroSD https://www.westerndigital.com/en-gb/products/memory-cards/sandisk-ultra-uhs-i-microsd
* Crucial X9 Pro 4TB Portable External SSD https://uk.crucial.com/products/ssd/crucial-x9-pro-ssd
* Geekworm 10mm Aluminum Alloy Heatsink https://geekworm.com/products/c296

## Features
* 4TB of storage.
* Automatic TLS Certificate renewal.
* Connect from your local network or via your personal domain.
* Automatic OS updates.
* Simple single user setup.

## Setup
* Use the Raspberry Pi Imager to setup a new bootable SD Card:
  * See: https://www.raspberrypi.com/documentation/computers/getting-started.html#raspberry-pi-imager
  * Choose the device: Raspberry Pi Zero.
  * Choose the OS: Raspberry Pi OS Lite (32-Bit).
  * Choose the storage.
  * Edit the OS customisation settings:
    * Set hostname: filetcloud
    * Set your username and password, ensuring a very strong password.
    * Configure the wireless LAN.
    * Enable SSH.
* Insert the SD Card.
* Power on the Raspberry Pi.
* Setup FiletCloud:
  * Connect to the Pi via SSH using your username: `ssh username@filetcloud.local`
  * Run the following command - note this will run admin commands on your device:
```bash
 wget https://raw.githubusercontent.com/fuglaro/filet-cloud/main/deployments/raspberry-pi-zero-2-w-ssd-autocert/setup -O - | sh
```
  * Enter your personal domain when prompted.
* Ensure your network is configured to forward ports 80, 443.
* You should now be able to access your Filet Cloud:
  * From any device on your home network: `https://filetcloud.local/`
  * Or remotely via your personal domain.
* To configure automatic device syncs, ensure your network is configured to forward port 22, and use a device backup tool such as Folder Sync Pro. Setting up and using ssh keys for this is recommended.
* If required set up ddclient to enable dynamic DNS (configure /etc/ddclient.conf, systemctl start ddclient.service).

## Metrics
* Idle power consumption: ~ 1.2W
* List folder speed: 15ms (tested with 8 entries over WiFi)
* Retrieve small file speed: 25ms (tested with 5KB file over WiFi)
* Retrieve big file wait time: 50ms (tested with 4.4MB JPEG over WiFi)
* Retrieve big file retrieval time: 13s (tested with 4.4MB JPEG over WiFi)
* Retrieve big file thumbnail wait time: 5s (tested with 4.4MB JPEG over WiFi)
* Retrieve big file thumbnail time: 0ms (tested with 4.4MB JPEG over WiFi)
