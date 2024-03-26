![kafka_exporter](https://socialify.git.ci/danielqsj/kafka_exporter/image?description=1&font=Inter&forks=1&pattern=Signal&stargazers=1&theme=Light)

kafka_exporter
==============

Kafka exporter for Prometheus. For other metrics from Kafka, have a look at the [JMX exporter](https://github.com/prometheus/jmx_exporter).

Table of Contents
-----------------

-	[Compatibility](#compatibility)
-	[Dependency](#dependency)
-	[Download](#download)
-	[Compile](#compile)
	-	[Build Binary](#build-binary)
	-	[Build Docker Image](#build-docker-image)
-	[Run](#run)
	-	[Run Binary](#run-binary)
	-	[Run Docker Image](#run-docker-image)
 -	-	[Run Docker Compose](#run-docker-compose)
-	[Flags](#flags)
    -	[Notes](#notes)
-	[Metrics](#metrics)
	-	[Brokers](#brokers)
	-	[Topics](#topics)
	-	[Consumer Groups](#consumer-groups)
-	[Grafana Dashboard](#grafana-dashboard)
-   [Contribute](#contribute)
-   [Donation](#donation)
-   [License](#license)

Compatibility
-------------

Support [Apache Kafka](https://kafka.apache.org) version 0.10.1.0 (and later).

Dependency
----------

-	[Prometheus](https://prometheus.io)
-	[Sarama](https://shopify.github.io/sarama)
-	[Golang](https://golang.org)

Download
--------

Binary can be downloaded from [Releases](https://github.com/grafana/kafka_exporter/releases) page.

Compile
-------

### Build Binary

```shell
make
```

### Build Docker Image

```shell
make docker
```

Docker Hub Image
----------------

```shell
docker pull grafana/kafka-exporter:latest
```

It can be used directly instead of having to build the image yourself. ([Docker Hub grafana/kafka-exporter](https://hub.docker.com/r/grafana/kafka-exporter)\)

Run
---

### Run Binary

```shell
kafka_exporter --kafka.server=kafka:9092 [--kafka.server=another-server ...]
```

### Run Docker Image

```
docker run -ti --rm -p 9308:9308 grafana/kafka-exporter --kafka.server=kafka:9092 [--kafka.server=another-server ...]
```

### Run Docker Compose
make a `docker-compose.yml` flie
```
services:
  kafka-exporter:
    image: danielqsj/kafka-exporter 
    command: ["--kafka.server=kafka:9092", "[--kafka.server=another-server ...]"]
    ports:
      - 9308:9308     
```
then run it
```
docker-compose up -d
```

Flags
-----

This image is configurable using different flags

| Flag name                      | Default        | Description                                                                                                                                    |
|--------------------------------|----------------|------------------------------------------------------------------------------------------------------------------------------------------------|
| kafka.server                   | kafka:9092     | Addresses (host:port) of Kafka server                                                                                                          |
| kafka.version                  | 2.0.0          | Kafka broker version                                                                                                                           |
| sasl.enabled                   | false          | Connect using SASL/PLAIN                                                                                                                       |
| sasl.handshake                 | true           | Only set this to false if using a non-Kafka SASL proxy                                                                                         |
| sasl.username                  |                | SASL user name                                                                                                                                 |
| sasl.password                  |                | SASL user password                                                                                                                             |
| sasl.mechanism                 |                | SASL mechanism can be plain, scram-sha512, scram-sha256                                                                                        |
| sasl.service-name              |                | Service name when using Kerberos Auth                                                                                                          |
| sasl.kerberos-config-path      |                | Kerberos config path                                                                                                                           |
| sasl.realm                     |                | Kerberos realm                                                                                                                                 |
| sasl.keytab-path               |                | Kerberos keytab file path                                                                                                                      |
| sasl.kerberos-auth-type        |                | Kerberos auth type. Either 'keytabAuth' or 'userAuth'                                                                                          |
| tls.enabled                    | false          | Connect to Kafka using TLS                                                                                                                     |
| tls.server-name                |                | Used to verify the hostname on the returned certificates unless tls.insecure-skip-tls-verify is given. The kafka server's name should be given |
| tls.ca-file                    |                | The optional certificate authority file for Kafka TLS client authentication                                                                    |
| tls.cert-file                  |                | The optional certificate file for Kafka client authentication                                                                                  |
| tls.key-file                   |                | The optional key file for Kafka client authentication                                                                                          |
| tls.insecure-skip-tls-verify   | false          | If true, the server's certificate will not be checked for validity                                                                             |
| server.tls.enabled             | false          | Enable TLS for web server                                                                                                                      |
| server.tls.mutual-auth-enabled | false          | Enable TLS client mutual authentication                                                                                                        |
| server.tls.ca-file             |                | The certificate authority file for the web server                                                                                              |
| server.tls.cert-file           |                | The certificate file for the web server                                                                                                        |
| server.tls.key-file            |                | The key file for the web server                                                                                                                |
| topic.filter                   | .*             | Regex that determines which topics to collect                                                                                                  |
| topic.exclude                  | ^$             | Regex that determines which topics to exclude                                                                                                  |
| group.filter                   | .*             | Regex that determines which consumer groups to collect                                                                                         |
| group.exclude                  | ^$             | Regex that determines which consumer groups to exclude                                                                                         |
| web.listen-address             | :9308          | Address to listen on for web interface and telemetry                                                                                           |
| web.telemetry-path             | /metrics       | Path under which to expose metrics                                                                                                             |
| log.enable-sarama              | false          | Turn on Sarama logging                                                                                                                         |
| use.consumelag.zookeeper       | false          | if you need to use a group from zookeeper                                                                                                      |
| zookeeper.server               | localhost:2181 | Address (hosts) of zookeeper server                                                                                                            |
| kafka.labels                   |                | Kafka cluster name                                                                                                                             |
| refresh.metadata               | 30s            | Metadata refresh interval                                                                                                                      |
| offset.show-all                | true           | Whether show the offset/lag for all consumer group, otherwise, only show connected consumer groups                                             |
| concurrent.enable              | false          | If true, all scrapes will trigger kafka operations otherwise, they will share results. WARN: This should be disabled on large clusters         |
| topic.workers                  | 100            | Number of topic workers                                                                                                                        |
| max.offsets                    | 1000           | Maximum number of offsets to store in the interpolation table for a partition                                                                  |

### Notes

Boolean values are uniquely managed by [Kingpin](https://github.com/alecthomas/kingpin/blob/master/README.md#boolean-values). Each boolean flag will have a negative complement:
`--<name>` and `--no-<name>`.

For example:

If you need to disable `sasl.handshake`, you could add flag `--no-sasl.handshake`

Metrics
-------

Documents about exposed Prometheus metrics.

For details on the underlying metrics please see [Apache Kafka](https://kafka.apache.org/documentation).

### Brokers

**Metrics details**

| Name            | Exposed information                    |
|-----------------|----------------------------------------|
| `kafka_brokers` | Number of Brokers in the Kafka Cluster |

**Metrics output example**

```txt
# HELP kafka_brokers Number of Brokers in the Kafka Cluster.
# TYPE kafka_brokers gauge
kafka_brokers 3
```

### Topics

**Metrics details**

| Name                                               | Exposed information                                 |
|----------------------------------------------------|-----------------------------------------------------|
| `kafka_topic_partitions`                           | Number of partitions for this Topic                 |
| `kafka_topic_partition_current_offset`             | Current Offset of a Broker at Topic/Partition       |
| `kafka_topic_partition_oldest_offset`              | Oldest Offset of a Broker at Topic/Partition        |
| `kafka_topic_partition_in_sync_replica`            | Number of In-Sync Replicas for this Topic/Partition |
| `kafka_topic_partition_leader`                     | Leader Broker ID of this Topic/Partition            |
| `kafka_topic_partition_leader_is_preferred`        | 1 if Topic/Partition is using the Preferred Broker  |
| `kafka_topic_partition_replicas`                   | Number of Replicas for this Topic/Partition         |
| `kafka_topic_partition_under_replicated_partition` | 1 if Topic/Partition is under Replicated            |

**Metrics output example**

```txt
# HELP kafka_topic_partitions Number of partitions for this Topic
# TYPE kafka_topic_partitions gauge
kafka_topic_partitions{topic="__consumer_offsets"} 50

# HELP kafka_topic_partition_current_offset Current Offset of a Broker at Topic/Partition
# TYPE kafka_topic_partition_current_offset gauge
kafka_topic_partition_current_offset{partition="0",topic="__consumer_offsets"} 0

# HELP kafka_topic_partition_oldest_offset Oldest Offset of a Broker at Topic/Partition
# TYPE kafka_topic_partition_oldest_offset gauge
kafka_topic_partition_oldest_offset{partition="0",topic="__consumer_offsets"} 0

# HELP kafka_topic_partition_in_sync_replica Number of In-Sync Replicas for this Topic/Partition
# TYPE kafka_topic_partition_in_sync_replica gauge
kafka_topic_partition_in_sync_replica{partition="0",topic="__consumer_offsets"} 3

# HELP kafka_topic_partition_leader Leader Broker ID of this Topic/Partition
# TYPE kafka_topic_partition_leader gauge
kafka_topic_partition_leader{partition="0",topic="__consumer_offsets"} 0

# HELP kafka_topic_partition_leader_is_preferred 1 if Topic/Partition is using the Preferred Broker
# TYPE kafka_topic_partition_leader_is_preferred gauge
kafka_topic_partition_leader_is_preferred{partition="0",topic="__consumer_offsets"} 1

# HELP kafka_topic_partition_replicas Number of Replicas for this Topic/Partition
# TYPE kafka_topic_partition_replicas gauge
kafka_topic_partition_replicas{partition="0",topic="__consumer_offsets"} 3

# HELP kafka_topic_partition_under_replicated_partition 1 if Topic/Partition is under Replicated
# TYPE kafka_topic_partition_under_replicated_partition gauge
kafka_topic_partition_under_replicated_partition{partition="0",topic="__consumer_offsets"} 0
```

### Consumer Groups

**Metrics details**

| Name                                         | Exposed informations                                                     |
|----------------------------------------------|--------------------------------------------------------------------------|
| `kafka_consumergroup_current_offset`         | Current Offset of a ConsumerGroup at Topic/Partition                     |
| `kafka_consumergroup_lag`                    | Current Approximate Lag of a ConsumerGroup at Topic/Partition            |
| `kafka_consumergroupzookeeper_lag_zookeeper` | Current Approximate Lag(zookeeper) of a ConsumerGroup at Topic/Partition |

#### Important Note

To be able to collect the metrics `kafka_consumergroupzookeeper_lag_zookeeper`, you must set the following flags:

* `use.consumelag.zookeeper`: enable collect consume lag from zookeeper
* `zookeeper.server`: address for connection to zookeeper

**Metrics output example**

```txt
# HELP kafka_consumergroup_current_offset Current Offset of a ConsumerGroup at Topic/Partition
# TYPE kafka_consumergroup_current_offset gauge
kafka_consumergroup_current_offset{consumergroup="KMOffsetCache-kafka-manager-3806276532-ml44w",partition="0",topic="__consumer_offsets"} -1

# HELP kafka_consumergroup_lag Current Approximate Lag of a ConsumerGroup at Topic/Partition
# TYPE kafka_consumergroup_lag gauge
kafka_consumergroup_lag{consumergroup="KMOffsetCache-kafka-manager-3806276532-ml44w",partition="0",topic="__consumer_offsets"} 1
```

### Consumer Lag

**Metric Details**

| Name                                 | Exposed information                                                          |
| ------------------------------------ | ---------------------------------------------------------------------------- |
| `kafka_consumer_lag_millis`          | Current approximation of consumer lag for a ConsumerGroup at Topic/Partition |
| `kafka_consumer_lag_extrapolation`   | Indicates that a consumer group lag estimation used extrapolation            |
| `kafka_consumer_lag_interpolation`   | Indicates that a consumer group lag estimation used interpolation            |

**Metrics output example**
```
# HELP kafka_consumer_lag_extrapolation Indicates that a consumer group lag estimation used extrapolation
# TYPE kafka_consumer_lag_extrapolation counter
kafka_consumer_lag_extrapolation{consumergroup="perf-consumer-74084",partition="0",topic="test"} 1
   
# HELP kafka_consumer_lag_interpolation Indicates that a consumer group lag estimation used interpolation
# TYPE kafka_consumer_lag_interpolation counter
kafka_consumer_lag_interpolation{consumergroup="perf-consumer-74084",partition="0",topic="test"} 1
   
# HELP kafka_consumer_lag_millis Current approximation of consumer lag for a ConsumerGroup at Topic/Partition
# TYPE kafka_consumer_lag_millis gauge
kafka_consumer_lag_millis{consumergroup="perf-consumer-74084",partition="0",topic="test"} 3.4457231197552e+10
```

Grafana Dashboard
-------

Grafana Dashboard ID: 7589, name: Kafka Exporter Overview.

For details of the dashboard please see [Kafka Exporter Overview](https://grafana.com/grafana/dashboards/7589-kafka-exporter-overview/).

Lag Estimation
-
The technique to estimate lag for a consumer group, topic, and partition is taken from the [Lightbend Kafka Lag Exporter](https://github.com/lightbend/kafka-lag-exporter). 

Once the exporter starts up, sampling of the next offset to be produced begins. The interpolation table is built from these samples, and the current offset for each monitored consumer group are compared against values in the table. If an upper and lower bound for the current offset of a consumer group are in the table, the interpolation technique is used. If only an upper bound is container within the table, extrapolation is used. 

For the lag computation, the number of offsets for each partition is trimmed down to `max.offsets` (default 1000), with the oldest offsets removed first.

Contribute
----------

To contribute to the upstream project, please open a [pull request](https://github.com/danielqsj/kafka_exporter/pulls).

To contribute to this fork please open a [pull request here](https://github.com/grafana/kafka_exporter/pulls)

Donation
--------

To donate to the developer of the project this is forked from please use the donation link below

![](https://github.com/danielqsj/kafka_exporter/raw/master/alipay.jpg)

License
-------

Code is licensed under the [Apache License 2.0](https://github.com/danielqsj/kafka_exporter/blob/master/LICENSE).
