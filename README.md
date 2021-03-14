# sysutils

This repository contains code to perform system level utilities. 
There are 2 programs available

## Installation

```
git clone https://github.com/ng5/sysutils && cd sysutils

# Use static link as preferred method for maximum portability
CGO_ENABLED=0 go install ./...
```

## testlistener
this open 3 ports on host machine to allow you to test connectivity.
```
~/go/bin/testlistener                  
2021/03/14 08:27:11 listening TCP: 12001
2021/03/14 08:27:11 listening UDP: 12002
2021/03/14 08:27:11 listening MULTICAST: 12003 (224.0.0.1)

```

## testnetwork
this allows to test network connectivity across 3 protocols: TCP, UDP & multicast.
Rules can be passed to this program with --file csv argument.
Sample rules.csv is provided.

### Basic connectivity from local machine
```
~/go/bin/testnetwork --file rules.csv
Description  HostName             Source               Target               Protocol     Status      
-------------------------------------------------------------------------------------------------------
ROW3         da09.ln.lan          localhost            google.com:80        TCP          OK          
ROW4         da09.ln.lan          localhost            google.com:443       TCP          OK          
ROW5         da09.ln.lan          localhost            da02:12002           UDP          OK          
ROW6         da09.ln.lan          localhost            da03:16379           TCP          OK          
ROW7         da09.ln.lan          127.0.0.1            da02:12002           UDP          OK          

```

### Testing from multiple machines
this is the most important part of this code. this program replicates itself to any number of source machines, 
runs itself on source machine and reports the result back to your terminal. You have the option to disable overwrite 
flag to run it faster. Disabling --overwrite will not copy executable on remote machine, this will save 
bandwidth and program will execute faster.

### below command will transfer itself and rule files on source machine and execute remotely.
files will be transferred to /tmp directory to avoid overwriting anything important on remote machines.
with --overwrite=false, files will not be transferred assuming they have been transferred before.
this will run test much faster.

```
~/go/bin/testnetwork --file=rules.csv --replicate=true --overwrite=true 
da02.ln.lan da01.csv 332 bytes copied
da02.ln.lan ~/go/bin/testnetwork 5104563 bytes copied
Description  HostName             Source               Target               Protocol     Status      
-------------------------------------------------------------------------------------------------------
ROW1         da02.ln.lan          da02.ln.lan          google.com:80        TCP          OK          
ROW2         da02.ln.lan          da02.ln.lan          google.com:443       TCP          OK          
ROW3         da02.ln.lan          da02.ln.lan          8.8.8.8:53           UDP          OK          
ROW4         da02.ln.lan          da02.ln.lan          4.4.4.4:53           UDP          OK  

```
