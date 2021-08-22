# Prometheus HTTP SD Server

## Description

The prom-http-sd-server allows users to dynamcially add/remove prometheus targets and labels to a target group and expose it to a [Prometheus HTTP SD](https://prometheus.io/docs/prometheus/latest/http_sd/) job.  All data is persisted on local disk.  Currently only the local data store is supported.

## Usage

Running the server:
```
./prom-http-sd-server -conf-path /path/to/config.yaml [-debug] [-version]
```

## Command Flags

`-conf-path` : The path to the configuration file to be used
`-debug` : Enable debug mode
`-version` : Show version and exit

## Configuration Options

`store_type` : The type of storage to use to persist the data.  Currently, only local supported
`store_path` : When using the `local` store_type, the path where to save the storage file.
`host` : The host on which to listen (default is 127.0.0.1)
`port`: The port on which to listen (default is 80)

## API Methods

### POST /api/target/<TARGET_GROUP>/<TARGET>
* Adds the new target to the specified target group

### DELETE /api/target/<TARGET_GROUP>/<TARGET>"
* Removes the target from the specified target group

### POST /api/labels/update/<TARGET_GROUP>?labels=<LABEL>=<VALUE>[&labels=<LABEL>=<VALUE>]
* Add one or more label/value pairs to the specified target group

### DELETE /api/labels/update/<TARGET_GROUP>/<LABEL_NAME>
* Delete the specified label from the target group

### GET /api/targets
* Return the list of targets (formated in expected HTTP SD format)

### GET /metrics
* Returns the list of prometheus metrics for the exporter

### GET /health
*  Returns the current health status of the exporter

### GET /debug_targets
* Returns the current list of targets along with the names of the target groups

### GET /debug_config
* Returns the current config which has been used to start the exporter


## Building

Run `make build` to build the the binary for the current operatory system or run `make build-all` to build for both Linux and OSX.   Refer to the makefile for additional options.


## License

Covered under the [MIT license](LICENSE.md).