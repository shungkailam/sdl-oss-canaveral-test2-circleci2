# start with alphine
FROM docker.dyn.ntnxdpro.com/ntnx-general-docker/alpine:3.7

RUN apk add --update-cache ca-certificates

# Create toolgroup and tooluser
RUN addgroup -S toolgroup -g 10001 && \
  adduser -D tooluser -u 10001 -G toolgroup

RUN mkdir -p /usr/src/app
WORKDIR /usr/src/app
# install the required software for tooling
RUN apk --update add python py-setuptools py-pip && \
    pip install --upgrade pip && \
    pip install elasticsearch-curator==5.7.6 && \
    pip install requests-aws4auth && \
    pip install certifi && \
    apk del py-pip && \
    rm -rf /var/cache/apk/*

COPY ./tool/curator.yaml ./curator.yaml

COPY ./tool/build/supportlogcleaner ./supportlogcleaner

COPY ./tool/build/softwareupdatecleaner ./softwareupdatecleaner

COPY ./tool/build/athenadatauploader ./athenadatauploader