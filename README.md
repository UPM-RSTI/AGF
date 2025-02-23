# AGF - Access Gateway Function 
Software Developed by [UPM RSTI Research group](https://blogs.upm.es/rsti).

<p align="justify">
This software has been developed for use in a virtualised environment, where virtual connections can be established with software that implements the different Network Functions of a 5G network core. It is based on two fundamental modules. One of them implements the main signalling protocols and functions between user equipment (UEs) and the core network functions with which they communicate (mainly, the AMF (Access and Mobility Management Function). The other allows a GTP-U tunnel (GPRS Tunnelling Protocol for user traffic) to be established from the software to the core function in charge of managing this type of traffic (UPF – User Plane Function). The traffic that can be injected through this tunnel emulates the real user traffic that would be generated in the user's mobile equipment, for which it opens an IP connection with one or more remote machines and waits to receive traffic from arbitrary network applications (ping, web traffic, streaming, etc.).
</p>

<p align="justify">
The characteristics of this program make it unnecessary to deploy a radio access network, as well as base stations, mobile equipment, etc. in test environments where experimentation on traffic behavior in the core of the network is required.
</p>

- **Main Branch**
  - **Description**: To be completed
  - **Status of develop**: To be completed
  - **5G Core compatibility**: Open5gs/free5gc

- **AGF_6Green Branch**
  - **Description**: To be completed
  - **Status of develop**: To be completed
  - **5G Core compatibility**: Open5gs/free5gc

- **6Green_Develop_Martin Branch**
  - **Depends** : AGF_6Green Branch
  - **Description**: To be completed
  - **Status of develop**: To be completed
  - **5G Core compatibility**: Open5gs/free5gc
 
- **6Green_Develop_Roberto Branch**
  - **Depends** : AGF_6Green Branch
  - **Description**: To be completed
  - **Status of develop**: To be completed
  - **5G Core compatibility**: Open5gs/free5gc

<p align="justify">
The repository structure consists of a file containing the main program code (stg-utg.go) and a directory (src) containing the rest of the program source code. Within this directory we can find the configuration file (config.yaml), editable by the user according to the characteristics of the use case where the program is going to be used, and directories containing the implementations of the program's functionalities. The stgutg directory contains the code of the modules that make up the software, divided into several files. In the other two folders (tglib and free5gclib), there are the libraries inherited from the Free5G project that implement the protocols and functions used by the program, on which modifications have been made to adapt them to the requirements of the SGT/UTI.
</p>

<p align="center">
<img src=code_structure.PNG width="600" height="300" />
</p>

# Index
- [AGF Dependencies](https://github.com/UPM-RSTI/AGF#AGF-Dependencies)
- [5G Core Installation](https://github.com/UPM-RSTI/AGF/blob/main/README.md#5g-core-installation)
- [AGF Installation](https://github.com/UPM-RSTI/AGF#AGF-Installation)
- [Configuration](https://github.com/UPM-RSTI/AGF#Configuraton)
- [Run](https://github.com/UPM-RSTI/AGF#Run)
- [Examples](https://github.com/UPM-RSTI/AGF#Examples)
- [FAQ](https://github.com/UPM-RSTI/AGF#FAQ)

## AGF Dependencies
```
go version: >= 1.20 (Tested with 1.20 and 1.21.4)
```
- Step 1: Download go.
Go to the official Go Downloads (https://go.dev/dl/) page and download the latest .tar.gz archive for Linux.

- Step 2: Step 2: Install Golang
```
sudo tar -C /usr/local -xzf go1.x.x.linux-amd64.tar.gz
```
(Replace go1.x.x with the actual version you downloaded.)

- Step 3: Set Up Environment Variables
Add Go’s bin directory to your PATH.

Type the following commands to export the Go Path persistent across reboots:
```
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export GOROOT=/usr/local/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin:$GOROOT/bin' >> ~/.bashrc
echo 'export GO111MODULE=auto' >> ~/.bashrc
```
Save the file and reload it:
```
source ~/.bashrc
```

- Step 4: Verify Installation
Check if Go is installed correctly:
```
go version
```


```
sudo apt-get install libpcap-dev
```

## 5G Core Installation
- [free5gc](https://github.com/UPM-RSTI/AGF/wiki/Core-5G-%E2%80%90-free5gc)
- [open5gs](https://github.com/UPM-RSTI/AGF/wiki/Core-5G-%E2%80%90-open5gs)
  
## AGF Installation
### Build executable

```
go build
```


## Configuration
The AGF configuration depends on the 5G network core you want to connect to (free5gc or open5gs). Below are two independent guides to modify the config.yaml file in each case.

### Specific configuration guide
```
nano src/config.yaml
```
- [AGF-free5gc](https://github.com/UPM-RSTI/AGF/wiki/AGF-%E2%80%90-Free5gc-configuration)
- [AGF-open5gs](https://github.com/UPM-RSTI/AGF/wiki/AGF-%E2%80%90-Open5gs-configuration)
  

## Run
### Run executable
```
./stgutgmain 
```
or
```
./stgutgmain -t 
```
for testing mode

---

## Examples
*To be complete

## FAQ

UPF container closes unexpectedly after docker compose up returning the following error: [ERRO][UPF][Main] UPF Cli Run Error: open Gtp5g: open link: create: operation not supported

--Try removing gtp5g current installation and reinstall again. Check [free5gc](https://github.com/UPM-RSTI/AGF/wiki/Core-5G-%E2%80%90-free5gc) for compatible versions and further explanation.   

![](logorsti.png) 








########## OLD README - Under Reconstruction #################

![](stgutg.png) 

# STGUTG

STGUTG (Signaling Traffic Generation/User Traffic Generation) is a software created for the generation of both signal and user traffic to be sent to a 5G network core. It is based on implementations from the [Free5GC](https://www.free5gc.org/) project and is distributed under an Apache 2.0 license.

Developed by [UPM RSTI Research group](https://blogs.upm.es/rsti).

In this repository, the software has been adapted for use with [Open5GS](https://open5gs.org/).



## Example: Deployment scenario with Open5GS

![](esquemagit.png)

This is a network scenario in which we are going to use Open5GS and the STGUTG to give Internet access to a virtual machine. The scenario consists of 3 VMs as it is represented in the picture. 

[Open5GS](https://open5gs.org/) is an open-source project for 5th generation (5G) mobile core networks, which intends to implement the 5G core network (5GC) defined in 3GPP Release 17 (R17). In this example, we use the NFs implemented in Open5GS to deploy a 5G core and then test the STGUTG software.


 


### 4. UE VM configuration 

Create an interface to reach the STGUTG VM and make a default route to make all the traffic reach the STGUTG. Set MTU to 1400B.


```
sudo ifconfig enp0s3 10.45.0.3 netmask 255.255.255.0 mtu 1400 up
```
```
sudo ip route add default via 10.45.0.4
```

### 5. Run the scenario

1. In Open5GS VM, execute in the Open5GS folder the following commands. This will start the NFs of the 5G core:

```
sudo systemctl restart open5gs-mmed
sudo systemctl restart open5gs-sgwcd
sudo systemctl restart open5gs-smfd
sudo systemctl restart open5gs-amfd
sudo systemctl restart open5gs-sgwud
sudo systemctl restart open5gs-upfd
sudo systemctl restart open5gs-hssd
sudo systemctl restart open5gs-pcrfd
sudo systemctl restart open5gs-nrfd
sudo systemctl restart open5gs-scpd
sudo systemctl restart open5gs-seppd
sudo systemctl restart open5gs-ausfd
sudo systemctl restart open5gs-udmd
sudo systemctl restart open5gs-pcfd
sudo systemctl restart open5gs-nssfd
sudo systemctl restart open5gs-bsfd
sudo systemctl restart open5gs-udrd
sudo systemctl restart open5gs-webui
```

2.  run the STGUTG software:
```
sudo ./stgutgmain
```

3. Use the UE VM to send traffic through the core to any Internet-based service (ping to 8.8.8.8 should suffice to test if the configuration is successful).

---
