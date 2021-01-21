module cloudservices

replace cloudservcies => ./

go 1.12

require (
	cloud.google.com/go v0.71.0 // indirect
	cloud.google.com/go/bigquery v1.10.0 // indirect
	cloud.google.com/go/storage v1.12.0 // indirect
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/DataDog/zstd v1.4.4 // indirect
	github.com/Shopify/sarama v1.22.2-0.20190604114437-cd910a683f9f // indirect
	github.com/VividCortex/gohistogram v1.0.0 // indirect
	github.com/alecthomas/participle v0.2.1
	github.com/alecthomas/repr v0.0.0-20181024024818-d37bc2a10ba1
	github.com/antihax/optional v1.0.0 // indirect
	github.com/apache/thrift v0.0.0-20151001171628-53dd39833a08 // indirect
	github.com/appscode/voyager v0.0.0-20200103113356-42d0a783b55f
	github.com/aws/aws-sdk-go v1.34.25
	github.com/aws/aws-sdk-go-v2 v0.24.0
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/cncf/udpa/go v0.0.0-20201001150855-7e6fe0510fb5 // indirect
	github.com/cockroachdb/cmux v0.0.0-20170110192607-30d10be49292
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/crossdock/crossdock-go v0.0.0-20160816171116-049aabb0122b // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dgraph-io/badger v1.5.3 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13 // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/eapache/go-resiliency v1.2.0 // indirect
	github.com/envoyproxy/go-control-plane v0.9.7 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.4.1 // indirect
	github.com/frankban/quicktest v1.7.3 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/analysis v0.19.7 // indirect
	github.com/go-openapi/errors v0.19.3
	github.com/go-openapi/loads v0.19.4
	github.com/go-openapi/runtime v0.19.11
	github.com/go-openapi/spec v0.19.6
	github.com/go-openapi/strfmt v0.19.4
	github.com/go-openapi/swag v0.19.7
	github.com/go-openapi/validate v0.19.6
	github.com/go-redis/redis v6.14.2+incompatible
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.4.3
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.4.2 // indirect
	github.com/gorilla/mux v1.7.4 // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/graarh/golang-socketio v0.0.0-20170510162725-2c44953b9b5f
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.12.2 // indirect
	github.com/hashicorp/go-hclog v0.14.0 // indirect
	github.com/hashicorp/go-plugin v1.3.0 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/go-version v1.1.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/yamux v0.0.0-20190923154419-df201c70410d // indirect
	github.com/iancoleman/strcase v0.1.2
	github.com/jaegertracing/jaeger v1.17.0
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jessevdk/go-flags v1.4.0
	github.com/jmoiron/sqlx v0.0.0-20180406164412-2aeb6a910c2b
	github.com/julienschmidt/httprouter v1.2.0
	github.com/kiali/k-charted v0.6.0
	github.com/klauspost/crc32 v0.0.0-20161016154125-cb6bfca970f6 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.2.0
	github.com/lyft/protoc-gen-star v0.5.2 // indirect
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/mattn/go-sqlite3 v2.0.1+incompatible // indirect
	github.com/mjibson/esc v0.2.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/olivere/elastic v6.2.27+incompatible
	github.com/opentracing-contrib/go-stdlib v0.0.0-20190519235532-cf7a6c988dc9 // indirect
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/pierrec/lz4 v2.4.1+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.12.0 // indirect
	github.com/prashantv/protectmem v0.0.0-20171002184600-e20412882b3a // indirect
	github.com/prometheus/client_golang v1.2.1
	github.com/prometheus/common v0.9.1 // indirect
	github.com/prometheus/procfs v0.0.10 // indirect
	github.com/qri-io/jsonschema v0.1.1
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563 // indirect
	github.com/rs/cors v1.7.0
	github.com/sectioneight/md-to-godoc v0.0.0-20161108233149-55e43be6c335 // indirect
	github.com/securego/gosec v0.0.0-20200203094520-d13bb6d2420c // indirect
	github.com/sha1sum/aws_signing_client v0.0.0-20170514202702-9088e4c7b34b
	github.com/spf13/afero v1.4.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.6.2 // indirect
	github.com/streadway/quantile v0.0.0-20150917103942-b0c588724d25 // indirect
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/thoas/go-funk v0.0.0-20180701190756-e2f8da694c9c
	github.com/uber/jaeger-client-go v2.22.1+incompatible // indirect
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	github.com/uber/tchannel-go v1.16.0 // indirect
	github.com/vektra/mockery v0.0.0-20181123154057-e78b021dcbb5 // indirect
	github.com/wadey/gocovmerge v0.0.0-20160331181800-b5bfa59ec0ad // indirect
	github.com/xeipuuv/gojsonschema v1.1.0
	github.com/xi2/httpgzip v0.0.0-20171230121455-d5b8e0abf572
	go.mongodb.org/mongo-driver v1.3.0 // indirect
	go.uber.org/atomic v1.5.1 // indirect
	go.uber.org/automaxprocs v1.3.0 // indirect
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/exp v0.0.0-20200224162631-6cc2880d07d6 // indirect
	golang.org/x/mod v0.3.1-0.20200828183125-ce943fd02449 // indirect
	golang.org/x/net v0.0.0-20201031054903-ff519b6c9102
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/sys v0.0.0-20201101102859-da207088b7d1 // indirect
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	golang.org/x/tools v0.0.0-20201103190053-ac612affd56b // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20201103154000-415bd0cd5df6 // indirect
	google.golang.org/grpc v1.33.1
	google.golang.org/protobuf v1.25.0
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/ini.v1 v1.52.0 // indirect
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.5.0 // indirect
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	helm.sh/helm/v3 v3.0.2
	k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/client-go v12.0.0+incompatible
	kmodules.xyz/client-go v0.0.0-20200105092743-4b797c0c0802 // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace (
	bitbucket.org/bertimus9/systemstat => ./vendor/bertimus9/systemstat
	// Transitive requirement from Helm.
	github.com/docker/docker => github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/haproxy_exporter => github.com/appscode/haproxy_exporter v0.7.2-0.20190508003714-b4abf52090e2
	github.com/xenolf/lego => github.com/appscode/lego v1.2.2-0.20181215093553-e57a0a1b7259
	gomodules.xyz/cert => ./vendor/gomodules.xyz/cert
	gomodules.xyz/jsonpatch/v2 => ./vendor/gomodules.xyz/jsonpatch/v2
	gomodules.xyz/version => ./vendor/gomodules.xyz/version
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
	k8s.io/api => k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191114105449-027877536833
	k8s.io/apimachinery => github.com/kmodules/apimachinery v0.0.0-20191119091232-0553326db082
	k8s.io/apiserver => github.com/kmodules/apiserver v0.0.0-20191119111000-36ac3646ae82
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191114101535-6c5935290e33
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191114113550-6123e1c827f7
	k8s.io/kubernetes => github.com/kmodules/kubernetes v1.17.0-alpha.0.0.20191127022853-9d027e3886fd
	vbom.ml/util => ./vendor/vbom/util
	github.com/stretchr/testify => github.com/stretchr/testify v1.6.1
)
