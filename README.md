# openGauss Server Exporter

Prometheus exporter for openGauss server metrics.

## Quick Start

This package is available for Docker:

```bash
# Start an example database
docker run --net=host -it --rm -e GS_PASSWORD=mtkOP@128 enmotech/opengauss
# Connect to it
docker run --net=host -e DATA_SOURCE_NAME="postgresql://postgres:password@localhost:5432/postgres?sslmode=disable" mogdb/opengauss_exporter
```

## Building and running

### gitee

The default make file behavior is to build the binary:

```bash
go clone https://gitee.com/opengauss/openGauss-prometheus-exporter.git
cd openGauss-prometheus-exporter
make build
export DATA_SOURCE_NAME="postgresql://login:password@hostname:port/dbname"
./bin/opengauss_exporter <flags>
```

To build the docker, run `make docker`.

### Flags

* `help`
  Show context-sensitive help (also try --help-long and --help-man).

* `web.listen-address`
  Address to listen on for web interface and telemetry. Default is `:9187`.

* `web.telemetry-path`
  Path under which to expose metrics. Default is `/metrics`.

* `disable-settings-metrics`
  Use the flag if you don't want to scrape `pg_settings`.

* `auto-discover-databases`
  Whether to discover the databases on a server dynamically.

* `config`
  Path to a YAML file containing queries to run. Check out [`og_exporter.yaml`](og_exporter_default.yaml)
  for examples of the format.

* `--dry-run`
  Do not run - print the internal representation of the metric maps. Useful when debugging a custom
  queries file.

* `constantLabels`
  Labels to set in all metrics. A list of `label=value` pairs, separated by commas.

* `version`
  Show application version.

* `exclude-databases`
  A list of databases to remove when autoDiscoverDatabases is enabled.

* `log.level`
  Set logging level: one of `debug`, `info`, `warn`, `error`, `fatal`

* `log.format`
  Set the log output target and format. e.g. `logger:syslog?appname=bob&local=7` or `logger:stdout?json=true`
  Defaults to `logger:stderr`.

### Environment Variables

The following environment variables configure the exporter:

* `OG_EXPORTER_URL` `PG_EXPORTER_URL` `DATA_SOURCE_NAME`
  the default legacy format. Accepts URI form and key=value form arguments. The
  URI may contain the username and password to connect with.

* `OG_EXPORTER_WEB_LISTEN_ADDRESS`
  Address to listen on for web interface and telemetry. Default is `:9187`.

* `OG_EXPORTER_WEB_TELEMETRY_PATH`
  Path under which to expose metrics. Default is `/metrics`.

* `OG_EXPORTER_DISABLE_SETTINGS_METRICS`
  Use the flag if you don't want to scrape `pg_settings`. Value can be `true` or `false`. Default is `false`.

* `OG_EXPORTER_AUTO_DISCOVER_DATABASES`
  Whether to discover the databases on a server dynamically. Value can be `true` or `false`. Default is `false`.

* `OG_EXPORTER_CONSTANT_LABELS`
  Labels to set in all metrics. A list of `label=value` pairs, separated by commas.

* `OG_EXPORTER_EXCLUDE_DATABASES`
  A comma-separated list of databases to remove when autoDiscoverDatabases is enabled. Default is empty string.

Settings set by environment variables starting with `OG_` will be overwritten by the corresponding CLI flag if given.

### Setting the openGauss server's data source name

The openGauss server's [data source name](http://en.wikipedia.org/wiki/Data_source_name)
must be set via the `OG_EXPORTER_URL` or `PG_EXPORTER_URL` or `DATA_SOURCE_NAME` environment variable.

Priorities are as follows

`OG_EXPORTER_URL` > `PG_EXPORTER_URL` > `DATA_SOURCE_NAME`

For running it locally on a default Debian/Ubuntu install, this will work (transpose to init script as appropriate):

  DATA_SOURCE_NAME="user=postgres host=/var/run/postgresql/ sslmode=disable" opengauss_exporter

Also, you can set a list of sources to scrape different instances from the one exporter setup. Just define a comma separated string.

  DATA_SOURCE_NAME="port=5432,port=6432" opengauss_exporter

See the [github.com/lib/pq](http://github.com/lib/pq) module for other ways to format the connection string.

> If you define connection strings for multiple databases, database version consistency is required
> export DATA_SOURCE_NAME=postgresql://gaussdb:password@127.0.0.1:26000/postgres?sslmode=disable,postgresql://gaussdb:password@127.0.0.1:26001/postgres?sslmode=disable

### Adding new metrics via a config file

The --config command-line argument specifies a YAML file containing additional queries to run.
Some examples are provided in [og_exporter.yaml](og_exporter_default.yaml).

### Automatically discover databases

To scrape metrics from all databases on a database server, the database DSN's can be dynamically discovered via the
`--auto-discover-databases` flag. When true, `SELECT datname FROM pg_database WHERE datallowconn = true AND datistemplate = false and datname != current_database()` is run for all configured DSN's. From the
result a new set of DSN's is created for which the metrics are scraped.

In addition, the option `--exclude-databases` adds the possibily to filter the result from the auto discovery to discard databases you do not need.

### run test

```shell
make build
cd test;sh test.sh ../bin/opengauss_exporter <config_file>
```

### OpenGauss

### Monitor user

```bash
CREATE USER dbuser_monitor with login monadmin PASSWORD 'Mon@1234';
grant usage on schema dbe_perf to dbuser_monitor;
grant select on pg_stat_replication to dbuser_monitor;

```

·

### primary and standby

```bash
docker network create opengauss_network --subnet=172.11.0.0/24
docker run --network opengauss_network --ip 172.11.0.101 \
  --privileged=true --name opengauss_primary  -h opengauss_primary  -p 1111:5432 -d \
  -e GS_PORT=5432 -e OG_SUBNET=172.11.0.0/24 -e GS_PASSWORD=Gauss@123 -e NODE_NAME=opengauss_primary \
  -e 'REPL_CONN_INFO=replconninfo1 = '\''localhost=172.11.0.101 localport=5434 localservice=5432 remotehost=172.11.0.102 remoteport=5434 remoteservice=5432'\''\n' enmotech/opengauss:1.1.0 -M primary
docker run --network opengauss_network --ip 172.11.0.102 \
  --privileged=true --name opengauss_standby1 -h opengauss_standby1 -p 1112:5432 -d \
  -e GS_PORT=5432 -e OG_SUBNET=172.11.0.0/24 -e GS_PASSWORD=Gauss@123 -e NODE_NAME=opengauss_standby1 \
  -e 'REPL_CONN_INFO=replconninfo1 = '\''localhost=172.11.0.102 localport=5434 localservice=5432 remotehost=172.11.0.101 remoteport=5434 remoteservice=5432'\''\n' enmotech/opengauss:1.1.0 -M standby
```
