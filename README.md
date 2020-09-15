TFGCo EventsGateway
===================

[![Build Status](https://travis-ci.org/topfreegames/eventsgateway.svg?branch=master)](https://travis-ci.org/topfreegames/eventsgateway)

## [Development README](#development-readme)
## [Client README](#client-readme)

<br/><br/>

# Client README

## Configuration

About client.NewClient(...) arguments:

* configPrefix: whatever comes just before `client` portion of your config
* config: a viper config with at least a `client` key holding Events Gateway settings
* logger: a logger.Logger instance
* client: should be nil for most cases, except for unit testing
* opts: extra []grpc.DialOption objects

`client` config format, with defaults:

```yaml
client:
  async: false # if you want to use the async or sync dispatch
  channelBuffer: 500 # (async-only) size of the channel that holds events
  lingerInterval: 500ms # (async-only) how long to wait before sending messages, in the hopes of filling the batch
  batchSize: 10 # (async-only) maximum number of messages to send in a batch
  maxRetries: 3 # (async-only) how many times to retry a dispatch if it fails
  retryInterval: 1s # (async-only) first wait time before a retry, formula => 2^retryNumber * retryInterval
  numRoutines: 2 # (async-only) number of go routines that read from events channel and send batches
  kafkatopic: default-topic # default topic to send messages
  grpc:
    serverAddress: localhost:5000
    timeout: 500ms
```

Code example:

```go
import (
  "context"

  "github.com/spf13/viper"
  "github.com/topfreegames/eventsgateway"
  "github.com/topfreegames/eventsgateway/logger"
)

func ConfigureEventsGateway() (*eventsgateway.Client, error) {
  config := viper.New() // empty Viper config
  config.Set("eventsgateway.client.async", true)
  config.Set("eventsgateway.client.kafkatopic", "my-client-default-topic")
  logger := &logger.NullLogger{} // Initialize you logger.Logger implementation here
  client, err := eventsgateway.NewClient("eventsgateway", config, logger, nil)
  if err != nil {
    return nil, err
  }
}

func main() {
  client, err := ConfigureEventsGateway()
  if err != nil {
    panic(err)
  }
  // here you pass along the context.Context you received,
  // DON'T pass just a context.Background() if you have a previous context.Context
  // Sync clients should handle errors accordingly
  err := client.Send(context.Background(), "event-name", map[string]string{"some": "value"})
  // Async clients error handling are transparent to the user 
  client.Send(context.Background(), "event-name", map[string]string{"some": "value"})
}

```

# Development README

## Running locally

All dependencies required to produce and consume events locally are bundled in this project.

1. `make deps-start` will start docker containers for `zookeeper` `kafka` and `localstack`.

These are the necessary dependencies for EventsGateway server.

2. `make run` starts EventsGateway server.

3. `make producer` executes a client that sends one dummy event.

4. `make spark-notebook` runs a jupyter-notebook container with a mounted notebook to consume from Kafka and write ORC files to S3.

Checkout the localhost address to access the Web UI over the container logs.

5. `make hive-start` starts hive stack containers necessary to create tables in hive-metastore and to query from a presto client.

### Bootstraping Localstack bucket and prefixes

After `make deps-start` and before `make gobblin` you'll need to bootstrap localstack's s3 to transfer data from kafka.

`aws --endpoint-url=http://localhost:4572 s3 mb s3://eventsgateway-local`

`aws --endpoint-url=http://localhost:4572 s3api put-bucket-acl --bucket eventsgateway-local --acl public-read-write`

`aws --endpoint-url=http://localhost:4572 s3api put-object --bucket eventsgateway-local --key output/sv-uploads-default-topic/daily/`

Note that `default-topic` should be replaced by the topic you're using in your client, that's the one used by `make testclient`.

### Creating topic table in Hive metastore

Run it inside `docker exec -it hive_hive-server_1 sh -c "/opt/hive/bin/beeline -u jdbc:hive2://localhost:10000"`.

To get the commands necessary to create database and table, run the respective cell at the end of `eventsgateway-streaming-orc` notebook.

After creation, you need to run `msck repair table table_name;` from hive server container to be able to query recent data.

### Querying with Presto

Install presto client, on mac `brew install presto`.

`presto --catalog hive --schema default`

presto:default >> `show tables;`

presto:default >> `select * from defaulttopic;`
