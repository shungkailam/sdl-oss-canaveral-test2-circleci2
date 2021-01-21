# Micro services organization
`common` folder contains all the shared utilities. It should not depend on other specific service folder except for the generated files.
`cloudmgmt` folder contains the REST service which acts as the public facing orchestrator.
`event` folder contains the event gRPC service.
