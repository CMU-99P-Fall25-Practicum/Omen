# Mininet-WiFi Raw Output files processing

The Raw Output module is responsible for transforming the the raw results from the test driver into usable input for the visualization module. Given a directory, this module will find the latest batch of results in the given path (by reading the timestamped subdirectories of the form YYYYMMDD_HHMMSS). It will coalesce the results into two files, placing them in a local `./results` directory.


# Usage
```shell
$ go run . <raw_file_directory>
```

## Output files

- ```pingall_full_data.csv``` : All ```pingall_full``` data after each movement.
    - Example file contents:
        ```csv
        data_type,movement_number,test_file,node_name,position,src,dst,tx,rx,loss_pct,avg_rtt_ms
        movement,1,test1.txt,sta1,"0,5,0",,,,,,
        movement,2,test2.txt,sta2,"0,-15,0",,,,,,
        movement,3,test3.txt,sta3,"10,10,0",,,,,,
        movement,4,test4.txt,sta4,"80,10,0",,,,,,
        ping,1,test1.txt,,,sta1,ap1,1,1,0,0.109
        ping,1,test1.txt,,,sta1,ap2,1,1,0,0.049
        ping,1,test1.txt,,,sta2,ap1,1,1,0,0.044
        ping,1,test1.txt,,,sta2,ap2,1,1,0,0.047
        ping,1,test1.txt,,,sta3,ap1,1,1,0,0.047
        ping,1,test1.txt,,,sta3,ap2,1,1,0,0.039
        ping,1,test1.txt,,,sta4,ap1,1,1,0,0.188
        ping,1,test1.txt,,,sta4,ap2,1,1,0,0.034
        ping,1,test1.txt,,,ap1,sta1,1,0,100,0
        ping,1,test1.txt,,,ap1,sta2,1,0,100,0
        ping,1,test1.txt,,,ap1,sta3,1,0,100,0
        ping,1,test1.txt,,,ap1,sta4,1,0,100,0
        ping,1,test1.txt,,,ap1,ap2,1,1,0,0.069
        ping,1,test1.txt,,,ap2,sta1,1,0,100,0
        ping,1,test1.txt,,,ap2,sta2,1,0,100,0
        ping,1,test1.txt,,,ap2,sta3,1,0,100,0
        ping,1,test1.txt,,,ap2,sta4,1,0,100,0
        ping,1,test1.txt,,,ap2,ap1,1,1,0,0.139
        ping,2,test2.txt,,,sta1,ap1,1,1,0,0.139
        ping,2,test2.txt,,,sta1,ap2,1,1,0,0.063
        ping,2,test2.txt,,,sta2,ap1,1,1,0,0.326
        ping,2,test2.txt,,,sta2,ap2,1,1,0,0.066
        ping,2,test2.txt,,,sta3,ap1,1,1,0,0.067
        ping,2,test2.txt,,,sta3,ap2,1,1,0,0.041
        ping,2,test2.txt,,,sta4,ap1,1,1,0,0.077
        ping,2,test2.txt,,,sta4,ap2,1,1,0,0.027
        ping,2,test2.txt,,,ap1,sta1,1,0,100,0
        ping,2,test2.txt,,,ap1,sta2,1,0,100,0
        ping,2,test2.txt,,,ap1,sta3,1,0,100,0
        ping,2,test2.txt,,,ap1,sta4,1,0,100,0
        ping,2,test2.txt,,,ap1,ap2,1,1,0,0.136
        ping,2,test2.txt,,,ap2,sta1,1,0,100,0
        ping,2,test2.txt,,,ap2,sta2,1,0,100,0
        ping,2,test2.txt,,,ap2,sta3,1,0,100,0
        ping,2,test2.txt,,,ap2,sta4,1,0,100,0
        ping,2,test2.txt,,,ap2,ap1,1,1,0,0.081
        ping,3,test3.txt,,,sta1,ap1,1,1,0,0.161
        ping,3,test3.txt,,,sta1,ap2,1,1,0,0.056
        ping,3,test3.txt,,,sta2,ap1,1,1,0,0.066
        ping,3,test3.txt,,,sta2,ap2,1,1,0,0.079
        ping,3,test3.txt,,,sta3,ap1,1,1,0,0.078
        ping,3,test3.txt,,,sta3,ap2,1,1,0,0.170
        ping,3,test3.txt,,,sta4,ap1,1,1,0,0.091
        ping,3,test3.txt,,,sta4,ap2,1,1,0,0.059
        ping,3,test3.txt,,,ap1,sta1,1,0,100,0
        ping,3,test3.txt,,,ap1,sta2,1,0,100,0
        ping,3,test3.txt,,,ap1,sta3,1,0,100,0
        ping,3,test3.txt,,,ap1,sta4,1,0,100,0
        ping,3,test3.txt,,,ap1,ap2,1,1,0,0.100
        ping,3,test3.txt,,,ap2,sta1,1,0,100,0
        ping,3,test3.txt,,,ap2,sta2,1,0,100,0
        ping,3,test3.txt,,,ap2,sta3,1,0,100,0
        ping,3,test3.txt,,,ap2,sta4,1,0,100,0
        ping,3,test3.txt,,,ap2,ap1,1,1,0,0.153
        ping,4,test4.txt,,,sta1,ap1,1,1,0,0.104
        ping,4,test4.txt,,,sta1,ap2,1,1,0,0.103
        ping,4,test4.txt,,,sta2,ap1,1,1,0,0.058
        ping,4,test4.txt,,,sta2,ap2,1,1,0,0.079
        ping,4,test4.txt,,,sta3,ap1,1,1,0,0.134
        ping,4,test4.txt,,,sta3,ap2,1,1,0,0.137
        ping,4,test4.txt,,,sta4,ap1,1,1,0,0.352
        ping,4,test4.txt,,,sta4,ap2,1,1,0,0.123
        ping,4,test4.txt,,,ap1,sta1,1,0,100,0
        ping,4,test4.txt,,,ap1,sta2,1,0,100,0
        ping,4,test4.txt,,,ap1,sta3,1,0,100,0
        ping,4,test4.txt,,,ap1,sta4,1,0,100,0
        ping,4,test4.txt,,,ap1,ap2,1,1,0,0.128
        ping,4,test4.txt,,,ap2,sta1,1,0,100,0
        ping,4,test4.txt,,,ap2,sta2,1,0,100,0
        ping,4,test4.txt,,,ap2,sta3,1,0,100,0
        ping,4,test4.txt,,,ap2,sta4,1,0,100,0
        ping,4,test4.txt,,,ap2,ap1,1,1,0,0.093
        ```
- ```final_iw_data.csv``` : Showcase final ```iw``` test result for each station.
    - Example file contents:
        ```csv
        device_type,test_file,device_name,interface,connected_to,ssid,freq,rx_bytes,rx_packets,tx_bytes,tx_packets,signal,rx_bitrate,tx_bitrate,bss_flags,dtim_period,beacon_int,flags,mtu,ether,tx_queue_len,rx_errors,rx_dropped,rx_overruns,rx_frame,tx_errors,tx_dropped,tx_overruns,tx_carrier,tx_collisions
        station,test5.txt,sta1,,02:00:00:00:04:00,test-ssid1,5180.0,343809,8714,4898,68,-51 dBm,54.0 MBit/s,6.0 MBit/s,short-slot-time,2,100,,,,,,,,,,,,,
        station,test5.txt,sta2,,02:00:00:00:04:00,test-ssid1,5180.0,343749,8713,4874,67,-69 dBm,54.0 MBit/s,6.0 MBit/s,short-slot-time,2,100,,,,,,,,,,,,,
        station,test5.txt,sta3,,02:00:00:00:04:00,test-ssid1,5180.0,159971,4070,2090,30,-69 dBm,54.0 MBit/s,6.0 MBit/s,short-slot-time,2,100,,,,,,,,,,,,,
        station,test5.txt,sta4,,02:00:00:00:05:00,test-ssid2,5180.0,341425,8684,4310,65,-69 dBm,54.0 MBit/s,6.0 MBit/s,short-slot-time,2,100,,,,,,,,,,,,,
        access_point,test5.txt,ap1,ap1-wlan1,,,,8598,137,11064,137,,,,,,,"UP,BROADCAST,RUNNING,MULTICAST",1500,02:00:00:00:04:00,,0,0,0,0,0,0,0,0,0
        access_point,test5.txt,ap2,ap2-wlan1,,,,5116,84,6628,84,,,,,,,"UP,BROADCAST,RUNNING,MULTICAST",1500,02:00:00:00:05:00,,0,0,0,0,0,0,0,0,0
        ```

## Example

Given the following directory structure:
```
./
├─ raw_output/
│  ├─ 20251001_201140
│  ├─ 20250908_090142
├─ other_file.txt
├─ other_directory/
```

Running `go run . ./raw_output` from within `cur/` will result in:

```
./
├─ results/
│  ├─ pingall_full_data.csv
│  ├─ final_iw_data.csv
├─ raw_output/
│  ├─ 20251001_201140
│  ├─ 20250908_090142
├─ other_file.txt
├─ other_directory/
```

with the results being pulled from `raw_output/20251001_201140`.