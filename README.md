# worker-incentives
Incentive payments to the workers

The program reads payment file (in csv format with address, amount) and pays the incentives on VideoCoin network in native VID.
The payments file is generated using the service from the following repo:    
https://github.com/videocoin/worker-availability  
Check the above repo for command line options.

After the file is generated and approved, the service in this repo can be used to process the payemnts. 

## Command to pay the worker incentives
```
./build/worker-incetives pay -c ./config.json  --input test.csv  --output testout.csv
```

### Description of commandline options

|Field|Descrption|Example|
|-----|------|-----|
|input|Input file in csv format|test_in.csv|
|output| output file in csv format|test_out.csv|
| c| Configuration file |config.json


### Config file
```
{
    "KeyFile": "keyfile_dev.json",
    "Password": "",
    "ClientID": ""
    "CredentialsFile": "credentials.json",
    "WorkerChainURL": "https://symphony.dev.videocoin.net/",
    "LogLevel": "trace" 
}
```

Description of the fields

|Field|Descrption|Example|
|-----|------|-----|
|KeyFile|Payer's wallet keyfile|keyfile.json|
|Password| Password for the payer's wallet||
|ClientID| GCP IAP Client ID||
|CredentialsFile| Google service credential file||
|WorkerChainURL| VideoCoin netwoork End Point|"https://symphony.dev.videocoin.net/|



