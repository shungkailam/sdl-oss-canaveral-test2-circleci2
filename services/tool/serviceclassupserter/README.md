# Service Class Upserter

The CLI  `/usr/src/cli/serviceclassupserter` in the `cloudmgmt` pod can be used to create/update/delete Service Classes in bulk.

It requires a data directory to be created which contains

1. `metadata.json` in the root of the data directory,
2. respective files in each their respective folders (not neccessary). e.g `kafka/my-first-kafka.json`. Each file contains the JSON for the Service Class creation REST API payload. The ID of the Service Class must be present. 

The `metadata.json` contains the list of files to be included by the upserter.

```
{ 
    "includes" : [
        "kafka/my-first-kafka.json",
        "istio/my-test-istio.json"
    ],
    "deleteMissingExcludes" : [
      "Prometheus"
    ]
}
```

The command by default runs in dry run mode. It can be disabled by specifiying ` -r ` option.

`/usr/src/cli/serviceclassupserter run --data-dir <data dir>`

A delete option can specified to delete the Service Classes which are present only in the cloud. Otherwise, they are ignored.

Service Class ID is the identifer used to match the Service Classes. Comparison is done to find out if the API call should be a POST (create) or an PUT (update).

A sample Kafka JSON file is 

```
{
  "id": "770171f3-b602-4f21-90a5-0917dc83786a",
  "type": "kafka",
  "svcVersion": "v100.0",
  "scope": "PROJECT",
  "minSvcDomainVersion": "1.15.0",
  "name": "Kafka",
  "description": "Message Streaming Service",
  "state": "FINAL",
  "bindable": true,
  "schemas": {
    "svcInstance": {
      "create": {
        "parameters": {
          "properties": {
            "profile": {
              "enum": [
                "Durability",
                "Performance"
              ],
              "type": "string"
            }
          },
          "title": "KafkaCreateOptions",
          "type": "object"
        }
      },
      "update": {
        "parameters": {
          "properties": {},
          "title": "KafkaUpdateOptions",
          "type": "object"
        }
      }
    },
    "svcBinding": {
      "create": {
        "parameters": {}
      }
    }
  }
}
```