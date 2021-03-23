# worker-incentives
Incentive payments to the workers

The program reads payment file (in csv format with address, amount) and pays the incentives on VideoCoin network in native VID

Config file
```
{
    "KeyFile": "keyfile_dev.json",
    "Password": "",
    "ClientID": ""
    "CredentialsFile": "credentials.json",
    "WorkerChainURL": "https://symphony.dev.videocoin.net/",
}
```

Description of the fields

|Field|Descrption|Syntax|
|-----|------|-----|
|KeyFile|Payer's wallet keyfile|keyfile.json|
|Password| Password for the wallet||
|ClientID| GCP IAP Client ID||
|CredentialsFile| Google service credential file||
|WorkerChainURL| VideoCoin netwoork End Point|"https://symphony.dev.videocoin.net/|




Command process the payments
```
./build/worker-incetives pay -c ./config.json  --input test.csv  --output testout.csv
```
Description of the options

|Field|Descrption|Syntax|
|-----|------|-----|
|input|Input file in csv format|test_in.csv|
|Password| output file in csv format|test_out.csv|

