TFGCo EventsGateway
===================

[![Build Status](https://travis-ci.org/topfreegames/eventsgateway.svg?branch=master)](https://travis-ci.org/topfreegames/eventsgateway)

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
