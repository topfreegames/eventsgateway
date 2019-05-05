TFGCo EventsGateway
===================

[![Build Status](https://travis-ci.org/topfreegames/eventsgateway.svg?branch=master)](https://travis-ci.org/topfreegames/eventsgateway)

## Running locally

All dependencies required to produce and consume events locally are bundled in this project.

1. `make deps-start` will start docker containers for `zookeeper` `kafka` and `localstack`.

These are the necessary dependencies for EventsGateway server.

2. `make run` starts EventsGateway server.

3. `make producer` executes a client that sends one dummy event.

4. `make gobblin` runs a gobblin job that consumes from sv-uploads-* kafka topics and creates partitioned avro files in s3 (localstack) eventsgateway-local.

5. `make hive-start` starts hive stack containers necessary to create tables in hive-metastore and to query from a presto client.

### Bootstraping Localstack bucket and prefixes

After `make deps-start` and before `make gobblin` you'll need to bootstrap localstack's s3 to transfer data from kafka.

`aws --endpoint-url=http://localhost:4572 s3 mb s3://eventsgateway-local`

`aws --endpoint-url=http://localhost:4572 s3api put-bucket-acl --bucket eventsgateway-local --acl public-read-write`

`aws --endpoint-url=http://localhost:4572 s3api put-object --bucket eventsgateway-local --key output/sv-uploads-default-topic/daily/`

Note that `default-topic` should be replaced by the topic you're using in your client, that's the one used by `make testclient`.

### Creating topic table in Hive metastore

Run it inside `docker exec -it hive_hive-server_1 sh -c "/opt/hive/bin/beeline -u jdbc:hive2://localhost:10000"`.

```
CREATE EXTERNAL TABLE `table_name` (
  `id` string COMMENT 'from deserializer', 
  `name` string COMMENT 'from deserializer', 
  `props` map<string,string> COMMENT 'from deserializer', 
  `servertimestamp` bigint COMMENT 'from deserializer', 
  `clienttimestamp` bigint COMMENT 'from deserializer')
PARTITIONED BY ( 
  `year` string, 
  `month` string, 
  `day` string)
ROW FORMAT SERDE 
  'org.apache.hadoop.hive.serde2.avro.AvroSerDe' 
WITH SERDEPROPERTIES ( 
  'avro.schema.literal'='\n{\n  \"namespace\": \"com.tfgco.eventsgateway\",\n  \"type\": \"record\",\n  \"name\": \"Event\",\n  \"fields\": [\n    {\n      \"name\": \"id\",\n      \"type\": \"string\"\n    },\n    {\n      \"name\": \"name\",\n      \"type\": \"string\"\n    },\n    {\n      \"name\": \"props\",\n      \"default\": {},\n      \"type\": {\n\t  \"type\": \"map\",\n\t  \"values\": \"string\"\n      }\n    },\n    {\n      \"name\": \"serverTimestamp\",\n      \"type\": \"long\"\n    },\n    {\n      \"name\": \"clientTimestamp\",\n      \"type\": \"long\"\n    }\n  ]\n}\n') 
STORED AS INPUTFORMAT 
  'org.apache.hadoop.hive.ql.io.avro.AvroContainerInputFormat' 
OUTPUTFORMAT 
  'org.apache.hadoop.hive.ql.io.avro.AvroContainerOutputFormat'
LOCATION
  's3a://eventsgateway-local/output/sv-uploads-default-topic/daily'
TBLPROPERTIES (
  'transient_lastDdlTime'='1551113653');
```

After creation, you need to run `msck repair table table_name;` from hive server container to be able to query recent data.

### Querying with Presto

Install presto client, on mac `brew install presto`.

`presto --catalog hive --schema default`

presto:default >> `show tables;`

presto:default >> `select * from defaulttopic;`
