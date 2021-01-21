# Kubernetes YAML Validation

Validation is done using the golang library github.com/xeipuuv/gojsonschema

In order to do this, we need kubernetes json schema.

Kubernetes uses Open API for its API spec.

For example, the v1.15.4 spec swagger-1.15.4.json is downloaded
from [here](https://raw.githubusercontent.com/kubernetes/kubernetes/v1.15.4/api/openapi-spec/swagger.json)

The gojsonschema requires json schema files for individual API resources.

The [kubernetes-json-schema project](https://github.com/instrumenta/kubernetes-json-schema)
maintains json schema files for individual k8s API resources generated from the swagger spec.

For example, the json schemas v1.15.4-standalone and v1.15.4-standalone-strict are downloaded from the above site.

There are several different flavors of the generated json schema: -standalone,
-local, -strict, etc. Previously we were using the -local version, but that
version although more compact, does not give the correct validation. To get
correct validation, we need to use the -standalone variants.

By default, we want to restrict k8s resources we support (e.g., for security
reason we don't support allowPrivilegeEscalation in security context). Thus we
need to modify the json schema.

Since the -standalone variants of the json schema expands all referenced object
definitions inline, it is very cumbersome to modify. (For example, podspec is
duplicated in deployment, stateful set, replica set, daemon set, job, etc.)

To make the modification more manageable, we will instead modify the
swagger.json and use the modified version to generate modified k8s json schemas.

The swagger-1.15.4-restricted.json is our modified, restricted version.

We the use the [tool](https://github.com/instrumenta/openapi2jsonschema) to
generate the json schemas.

Unfortunately there are some issue in the tool causing the generation to hang.

To work around it, I made a local copy of the tool in scripts/ and modified it
to avoid the hang. We can then use scripts/run.sh to generate the schema.

The generated schema is in v1.15.4-standalone-strict-restricted-full
We further pick from it a subset of resources we want to restrict to in
v1.15.4-standalone-strict-restricted

