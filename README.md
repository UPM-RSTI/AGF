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




