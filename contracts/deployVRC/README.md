## Utility to Deploy Validator Registraction Contract

This is a utility to help users deploy validator registration contract for running their own beacon chain node in a local containerized set up. To run the utility, it assumes there is a running geth node as a separate process attached to proof-of-work main chain. The utility will deploy the validator registration contract and print out the contract address. Users will pass the contract address to the beacon chain node to monitor when they have been conducted to become an active validator.

### Usage

*Name:*  
   **deployVRC** - this is a util to deploy validator registration contract

*Usage:*  
   deployVRC [global options] command [command options] [arguments...]

*Flags:*  
   **--dataDirPath**    Data directory for the databases and keystore (default: "./datadir")  
   **--ipcPath**        Filename for IPC socket/pipe within the datadir (default: "./geth.ipc")      
   **--passwordFile**   Password file for unlock account (default: "./password.txt")   
   **--help, -h**            show help   
   **--version, -v**         print the version   

### Example
./deployVRC  --ipcPath /path/to/your/geth.ipc --dataDirPath /path/to/your/dataDir --passwordFile /path/to/your/password.txt


### Output
INFO[0001] New contract deployed at 0x5275C2220C574330E230bFB7e4a0b96f60a18f02 
