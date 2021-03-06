# Prometheus HTTP SD Server

## Description

The prom-http-sd-server allows users to dynamcially add/remove prometheus targets and labels to a target group and expose it to a [Prometheus HTTP SD](https://prometheus.io/docs/prometheus/latest/http_sd/) job.  At this time, only the **local** and **consul** data stores are supported although there are plans to add others in the near future.


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

### Targets

* **GET /api/targets**
    * Return the list of targets (formated in expected HTTP SD format)
* **POST /api/target/<TARGET_GROUP>/<TARGET>**
    * Adds the new target to the specified target group
* **DELETE /api/target/<TARGET_GROUP>/<TARGET>**
    * Remove the target from the specified target group
* * **DELETE /api/target/<TARGET_GROUP>**
    * Delete a given target group along with all of its hosts and labels

### Labels

* **GET /api/labels/<TARGET_GROUP>**
    * Get the list of labels for a given target group
* **POST /api/labels/update/<TARGET_GROUP>?labels=<LABEL>=<VALUE>[&labels=<LABEL>=<VALUE>]**
    * Add one or more label/value pairs to the specified target group
* **DELETE /api/labels/update/<TARGET_GROUP>/<LABEL_NAME>**
    * Delete the specified label from the target group

### Miscelaneous

* **GET /metrics**
    * Return the list of prometheus metrics for the exporter
* **GET /health**
    *  Return the current health status of the exporter
* **GET /debug_targets**
    * Return the current list of targets along with the names of the target groups
* **GET /debug_config**
    * Return the current config which has been used to start the exporter


## Available Data Stores

Currently, the following data stores are available although others are planned to be added in the near future:

* local : Uses a local-disk based file backed by BoltDB
* consul : Uses consul as the data store via the KV API.  Please note the consul KV store has a default key value size limit of 512KB. (See [this](https://www.consul.io/docs/troubleshoot/faq#q-what-is-the-per-key-value-size-limitation-for-consul-s-key-value-store))


## Building

### 1. Checkout required code version

First, ensure you have checked out the proper release tag in order to get all files/dependencies corresponding to that version. 

### 2. Build Go binary

Run `make build` to build the the binary for the current operatory system or run `make build-all` to build for both Linux and OSX.   Refer to the makefile for additional options.

### 3. Build Docker container
Run the following docker command to build the image
```
docker build -t $(cat VERSION.txt) --build-arg VERSION=$(cat VERSION.txt) .
```


## License

Covered under the [MIT license](LICENSE.md).

## Author

Alain Lefebvre <hartfordfive 'at' gmail.com>