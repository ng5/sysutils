# sysutils

This repository contains code to perform system level utilities. There are 2 programs available

## Installation

```
git clone https://github.com/ng5/sysutils && cd sysutils

# Use static link as preferred method for maximum portability
CGO_ENABLED=0 go install ./...
```

## testlistener

if you don't have any server applications yet, use this program to open 3 ports on host machine to allow you to test
connectivity.

```
~/go/bin/testlistener                  
2021/03/14 19:28:50 listening TCP: 12001
2021/03/14 19:28:50 listening UDP: 12002
2021/03/14 19:28:50 listening MULTICAST: 239.0.0.1:12003
```

## testnetwork

this allows to test network connectivity across 3 protocols: TCP, UDP & multicast. Rules can be passed to this program
with --file csv argument. Sample rules.csv is provided.

### Basic connectivity from local machine

```
~/go/bin/testnetwork --file rules.csv
Description  HostName             Source               Target               Protocol     Status      
-------------------------------------------------------------------------------------------------------
ROW5         da09.ln.lan          localhost            google.com:80        TCP          OK          
ROW6         da09.ln.lan          localhost            google.com:443       TCP          OK          
ROW7         da09.ln.lan          localhost            8.8.8.8:53           UDP          OK          
ROW8         da09.ln.lan          localhost            4.4.4.4:53           UDP          OK   
```

### Testing from multiple machines

this is the most important part of this code. this program replicates itself to any number of source machines, runs
itself on source machine and reports the result back to your terminal. You have the option to disable overwrite flag to
run it faster. Disabling --overwrite will not copy executable on remote machine, this will save bandwidth and program
will execute faster.

### below command will transfer itself (testnetwork executable) and run rule file on remote machine.

files will be transferred to /tmp directory to avoid overwriting anything important on remote machines. with
--overwrite=false, files will not be transferred provides they have been transferred earlier. this will run tests much
faster.

```
~/go/bin/testnetwork --file=rules.csv --remote=true --overwrite=true 
da02.ln.lan rules.csv 376 bytes copied
da02.ln.lan ~/go/bin/testnetwork 5104563 bytes copied
Description  HostName             Source               Target               Protocol     Status      
-------------------------------------------------------------------------------------------------------
ROW1         da01.ln.lan          da01.ln.lan          google.com:80        TCP          OK          
ROW2         da01.ln.lan          da01.ln.lan          google.com:443       TCP          OK          
ROW3         da01.ln.lan          da01.ln.lan          8.8.8.8:53           UDP          OK          
ROW4         da01.ln.lan          da01.ln.lan          4.4.4.4:53           UDP          OK          
ROW5         da01.ln.lan          localhost            google.com:80        TCP          OK          
ROW6         da01.ln.lan          localhost            google.com:443       TCP          OK          
ROW7         da01.ln.lan          localhost            8.8.8.8:53           UDP          OK          
ROW8         da01.ln.lan          localhost            4.4.4.4:53           UDP          OK    
```
