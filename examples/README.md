# Ignite Go Client Examples
This package contains examples that help you to get accustomed to the Ignite Go client features and how to use them.

## How to run Examples?
Some of Go Client Examples requires specific Java classes (such as filters or listeners) to be present on Ignite server side.
They are stored as a separate Maven project located in `ignite-go-client/internal/testing/java`. 
Make sure you built them before start playing with the Examples. You can do this by running the following command from
the root directory of the project:
```shell
make build-java
```

All Ignite Go Client Examples requires Ignite cluster to be up and running locally. Ignite Go Client used in examples
will try to connect to Ignite cluster using the default address - localhost:10800.

You can start preconfigured Ignite cluster consisting of 3 nodes using the following command: 
```shell
docker compose -f docker/docker-compose.yml up
```

To run a particular example navigate to the folder of the example you are interested in and run the following command:
```shell
go run -tags=testing .
```