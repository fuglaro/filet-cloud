# filet-cloud
A lean slice of Cloud Storage

This project attempts to make a sophisticated personal cloud storage solution similar to the google-drive ecosystem, using the design philosophies of the filet project series. This project, like others in the series, aggressively pushes for code minimalism and the essence of simplicity.

This main interface to the data is via an SFTP server over a standard SSH connection. A surprising number of clients support this interface and it's a big productivity win to be able to use things like rsync and ssh. Account management is as simple as using Linux user accounts and ssh authentication. A core part of this project is the SFTP webserver also from the filet project series (https://github.com/fuglaro/filet-cloud-web) which gives access via a webpage.

It targets a build running on a Raspberry Pi.

## Hardware
The following hardware was used for this build:
* Raspberry Pi 4B 4GB https://www.raspberrypi.org/products/raspberry-pi-4-model-b/
* 32GB Transcend microSDXC/SDHC 300S https://www.transcend-info.com/Products/No-948
* Seagate 5TB Basic Portable External Drive https://www.seagate.com/gb/en/products/external-hard-drives/basic-external-hard-drive/
* 2.7inch Mono E-Ink display (with 4 buttons) https://www.waveshare.com/wiki/2.7inch_e-Paper_HAT

## Setup
### Basic Host Setup
* Install Raspberry Pi OS to a microSD card (https://www.raspberrypi.org/software/)
* Enable ssh on your Pi (https://www.raspberrypi.org/documentation/remote-access/ssh/)
* Make a better password for the pi user (https://www.raspberrypi.org/documentation/configuration/security.md)
* Set up WiFi if needed (https://www.raspberrypi.org/documentation/configuration/wireless/).
* If you intend to connect from outside your local network, setup port forwarding for resired ports (https://en.wikipedia.org/wiki/Port_forwarding), static DHCP, and dynamic DNS, if needed (https://wiki.archlinux.org/title/Dynamic_DNS).
  * Port 22: SFTP and full SSH access - if you wan't to only allow SFTP, then configure OpenSSH (https://wiki.archlinux.org/title/SFTP_chroot)
  * Port 80 and 443: Web UI access via TLS - Port 80 is only open for TLS certificate renewal.
* Ensure you have connected power, network, an empty USB drive to store the data, inserted the SD Card, and disconnected all other USB drives, then power on.

### USB Drive Setup
* Format the USB drive (for the data) as btrfs (https://wiki.archlinux.org/title/Btrfs).

### Filet-Cloud Installation
```bash
ssh pi@raspberrypi.local
git clone https://github.com/fuglaro/filet-cloud.git
sudo filet-cloud/install
```
Stay logged in for the remaining setup.

### Encrypted Connections - TLS (HTTPS) Setup
If you intend to connect from outside a trusted network including through port forwarding, you will need to set up digital certificates for HTTPS connections.
```bash
sudo filet-cloud/install_certs
```

### Create New Login Account:
```bash
filet-cloud-new-user
```
## Compatible Clients
* Android filebrowser - Solid Explorer
* Android filebrowser (opensource) - Ghost Commander (with SFTP plugin)
* Android filesyncer - FolderSync
* Linux filebrowser client - Filezilla

## Future work
* Full feature list.

## Design and Engineering Philosophies

This project explores how far a software product can be pushed in terms of simplicity and minimalism, both inside and out, without losing powerful features. Web programs and cloud tools tends to be bloated and buggy, as all software tends to be. *filetcloud* pushes a personal cloud solution to its leanest essence. It is a joy to use because it does what it needs to, reliably and quickly, and tries to do nothing else. The opinions that drove the project are:

* **Complexity must justify itself**.
* Lightweight is better than heavyweight.
* Select your dependencies wisely: they are complexity, but not using them, or using the wrong ones, can lead to worse complexity.
* Powerful features are good, but simplicity and clarity are essential.
* Adding layers of simplicity, to avoid understanding something useful, only adds complexity, and is a trap for learning trivia instead of knowledge.
* Steep learning curves are dangerous, but don't just push a vertical wall deeper; learning is good, so make the incline gradual for as long as possible.
* Allow other tools to thrive - e.g: terminals don't need tabs or scrollback, that's what tmux is for.
* Fix where fixes belong - don't work around bugs in other applications, contribute to them, or make something better.
* Improvement via reduction is sometimes what a project desperately needs, because we do so tend to just add. (https://www.theregister.com/2021/04/09/people_complicate_things/, https://www.nature.com/articles/s41586-021-03380-y)

# Thanks to, grateful forks, and contributions

We stand on the shoulders of giants. They own this, far more than I do.

* https://github.com/fuglaro/filet-cloud-web
* https://www.raspberrypi.org
* https://www.python.org/
* https://www.gnu.org/software/bash/
* https://www.transcend-info.com
* https://www.seagate.com
* https://www.waveshare.com
* https://wiki.archlinux.org
* https://en.wikipedia.org
* https://www.theregister.com
* https://www.nature.com/articles/s41586-021-03380-y
