# Mininet-WiFi Raw Output files processing

The Raw Output module is responsible for transforming the the raw results from the test driver into usable input for the visualization module. Given a directory, this module will find the latest batch of results in the given path (by reading the timestamped subdirectories of the form YYYYMMDD_HHMMSS). It will coalesce the results into two files, placing them in a local `./results` directory.


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