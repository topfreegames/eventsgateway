TFGCo EventsGateway
===================

## [Development README](#development-readme)
## [Client README](#client-readme)

<br/><br/>

# Server README

The server and client package are isolated. There are no direct dependencies between them.
All server related packages are under the `server/` folder.
All server components should run on docker, including the server API execution.

## Configuration

All config can be found on [config/local.yaml](config/local.yaml).

By default, the API connects with Kafka via docker network service.
Since Kafka advertises its ip via docker service, its necessary that the API runs in docker as well.

```yaml
kafka:
  producer:
    brokers: kafka:9092
```

# Development

## Running locally

All dependencies required to produce and consume events locally are bundled in this project.

1. `cd server/` folder.
2. `make build-dev` builds a docker image to run the API in development mode, and run tests.
3. `make deps-start` starts all dependencies: zookeeper, kafka and jeager.
4. `make run` starts the eventsgateway API in a docker container in development mode (reading from local files). Exposes ports 5000 (api) and 6060 (pprof).
5. `cd ..` to get back to the client 0package.
6. `make producer` executes a client that sends one dummy event. Execute it as many times as you want. It will be the number of rows inserted in the table.
7. `make spark-notebook` runs a jupyter notebook with a pyspark script to consume events from Kafka and store it as a Delta table.

## Testing

To test the server package execute `make test`