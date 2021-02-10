# KPS Connector Go SDK
This is the Golang SDK for building KPS Connectors.

## Overview
A KPS Connector is a kubernetes application that implements a specific GRPC contract. The contract is described in
the [kps-connector-idl](github.com/nutanix/kps-connector-idl) repository which has also been submoduled in this repository. Particularly,
the service has to implement three methods in order to fulfil the Connector contract.

```proto
service ConnectorService {
    // GetPayload should return all payloads given a payload kind:
    //   - If payload kind is set to STREAM, it should return all available streams (including discovered streams)
    //   - If payload kind is set to CONFIG, it should return the current config in use.
    rpc GetPayload(GetPayloadRequest) returns (GetPayloadResponse);

    // SetPayload takes a connector ID and a list of payloads and applies those
    // payloads to the relevant connector:
    //   - Payloads of kind STREAM should be subscribed to by the connector
    //   - Payloads of kind CONFIG should be used to update the current connector config
    rpc SetPayload(SetPayloadRequest) returns (SetPayloadResponse);

    // GetEvents returns all the events for a given connector ID. Events can be of type Alert or Status
    rpc GetEvents(GetEventsRequest) returns (GetEventsResponse);
}
```
This SDK provides generated golang stubs along with convenience libraries to build KPS connectors in Golang.

## Terminology Primer
### KPS Concepts
#### Project
Project is a KPS abstraction which corresponds to a multi-service-domain Kubernetes namespace.
#### Category
Category is a KPS abstraction which can be used as a selector for resources such as service domains or streams.
#### Service Domain
A service domain refers to a single-node or multi-node deployment of KPS.
#### Data Pipeline
A data pipeline is real time stream processing solution on KPS which can take input from connector streams, transform the data received from streams via functions, and output data into connector streams.
#### Function
A function is a transformation that can be used to transform data in a data pipeline.

### KPS Connector Concepts
#### Class
Class is a recipe that defines the type of connector. The Class is defined as a JSON file and contains the following properties:
- name: name of the connector class
- type: whether the class supports INGRESS, EGRESS, or is BIDIRECTIONAL
- staticParameterSchema: JSON schema of the properties that will be used as template parameters during the instance creation.
In the example below, note how the `image_tag` is used to set the docker image tag at instance creation.
- configParameterSchema: JSON schema of the connector config 
- streamParameterSchema: JSON schema of the connector stream

Here is an example of a `class.json`
```json
{
  "name": "natsconnector",
  "description": "This is a class definition of NATS data connector.",
  "connectorVersion": "1.0",
  "minSvcDomainVersion": "3.0.0",
  "type": "BIDIRECTIONAL",
  "staticParameterSchema": {
    "type": "object",
    "properties": {
      "image_tag": {
        "type": "string",
        "description": "test docker image tag to render in yaml"
      }
    }
  },
  "configParameterSchema": {
    "type": "object",
    "properties": {
      "log_level": {
        "type": "string",
        "description": "connector docker container log level"
      }
    }
  },
  "streamParameterSchema": {
    "type": "object",
    "description": "Stream schema",
    "properties": {
      "subject": {
        "type": "string",
        "description": "subject to fetch the messages from / emit the messages to"
      },
      "broker": {
        "type": "string",
        "description": "address of the NATS broker to read from / write to"
      }
    }
  },
  "yamlData": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: natsconnector\nspec:\n  replicas: 1\n  selector:\n    matchLabels:\n      app: natsconnector\n  template:\n    metadata:\n      name: natsconnector\n      labels:\n        app: natsconnector\n    spec:\n      containers:\n        - name: natsconnector\n          image: \"770301640873.dkr.ecr.us-west-2.amazonaws.com/edgecomputing/connector/natsconnector:{{ .Parameters.image_tag }}\"\n          imagePullPolicy: Always\n          securityContext:\n            runAsUser: 9999\n            allowPrivilegeEscalation: false\n          ports:\n            - containerPort: 8000\n---\nkind: Service\napiVersion: v1\nmetadata:\n  name: natsconnector-svc\nspec:\n  selector:\n    app: natsconnector\n  ports:\n    - protocol: TCP\n      name: natsconnector\n      port: 9000\n      targetPort: 8000\n"
}
```

The connector class can be registered with KPS using the CLI tool
```
kps create connectorclass -f class.json
```

#### Instance
Instance is an instance of a connector deployed on a project namespace. For example, A NATS Connector class can be used to create a NATS connector instance on a project.
Creating an instance requires the class and the values for static parameters of the class.
Here is an example of a `instance.json`
```json
{
  "name": "natsconnector",
  "connectorClassID": "107ebd6c-6ae8-4830-a91a-44d2425b575b",
  "projectId": "577a55d4-5115-495f-990d-36244af09724",
  "staticParameters": {
    "image_tag": "latest"
  }
}
```

The connector instance can be created using the CLI tool
```
kps create connectorinstance -f instance.json
```
#### Stream
Stream is a singular unit of connection. It defines connection data needed for a connector to connect to the underlying resource.
A stream that is designated for bringing data into a data pipeline has the direction `INGRESS` and a stream designated to move
data out of a data pipeline has the direction `EGRESS`. For example, A NATS connector stream for moving data from a NATS
subject into a data pipeline will define all the information needed to connect to a NATS subject. Here is an example of a `stream.json`
```json
{
  "name": "natsconnectorstream-in",
  "connectorInstanceID": "f29673b0-10f4-44dc-b4fd-02331ace2ffa",
  "labels": [
    {
      "id": "412c01ae-7fa9-442e-b102-1964c7788214",
      "value": "nats"
    }
  ],
  "direction": "INGRESS",
  "serviceDomainIds": [
    "fe1ec650-d41e-4ecb-9d75-da94089deb7e"
  ],
  "stream": {
    "subject": "nats_in",
    "broker": "nats://nats:4222"
  }
}
```
The connector stream can be created using the CLI tool
```
kps create connectorstream -f stream.json
```
#### Config
Config is a mechanism for receiving runtime configuration updates to the connector instance. Here's an example of `config.json`
```json
{
  "name": "natsconnectorsconfig",
  "description": "This is a dynamic config for nats connector instance",
  "connectorInstanceID": "f29673b0-10f4-44dc-b4fd-02331ace2ffa",
  "config": {
    "log_level": "INFO"
  },
  "serviceDomainIds": [
    "fe1ec650-d41e-4ecb-9d75-da94089deb7e"
  ]
}
```
The connector config can be created using the CLI tool
```
kps create connectorconfig -f config.json
```
#### Event
Event is a mechanism to propagate events such as status and alerts on behalf of the connector. Events emitted by the connector
can be viewed using the CLI
```
kps get connectorevents -i f29673b0-10f4-44dc-b4fd-02331ace2ffa
```

## Go Connector SDK 
This repository contains the Go libraries that can be used to build connectors that can interface with
Data Pipelines in Nutanix Karbon Platform Services.

The library consists of the following Go packages:
- connector: Contains the generated Go protobuf and grpc stubs for service contract defined in  [kps-connector-idl](github.com/nutanix/kps-connector-idl). 
- transport: Contains the `Client` that can publish data to and subscribe data from the streams defined in KPS.
- events: Contains the `Registry` and `Event` constructors that can be used for emitting events such as status and alerts for the Connector.

## Quick Start
The fastest way to build your own Connector is by building one using our Golang Connector Template. The template is an
opinionated implementation of a KPS connector written using the Golang Connector SDK with ease of modificaiton in mind.

You can find the connector template at [kps-connector-go-template](https://github.com/nutanix/kps-connector-go-template)
and build template based connector in Golang by following the instructions in README.md on the template repo.

## FAQ
Q: What Service Domain version do I need to be on to use KPS Connectors<br/>
A: KPS Connectors require KPS Service Domain version 2.3.0 to work

Q: When to write your own Connector<br/>
A: Nutanix provides a library of connectors made by Nutanix. If you don't find a connector that you need in the library, you can build a new connector.

### Questions, issues or suggestions?
Reach us at karbon-platform-services-api@nutanix.com or file an issue on the Github repository.
