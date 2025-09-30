# Mininet-WiFi Raw Output files processing
## How to start
Execute the following command to start processing raw input files
```shell
$ go run . <raw_file_directory>
```
For example
```shell
$ go run . ../1_spawn_topology/mn_result_raw
```
It will find the file with the latest sundirectory, and extract information within to create two ```.csv``` files in the ```./result``` directory.
- ```pingall_full_data.csv``` : All ```pingall_full``` data after each movement.
- ```final_iw_data.csv``` : Showcase final ```iw``` test result for each station.