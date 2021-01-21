# start with alphine
FROM docker.dyn.ntnxdpro.com/ntnx-general-docker/alpine:3.7

# Create tenantpoolgroup and tenantpooluser
RUN addgroup -S tenantpoolgroup -g 10001 && \
  adduser -D tenantpooluser -u 10001 -G tenantpoolgroup
RUN apk add --update-cache ca-certificates

RUN mkdir -p /usr/src/app/
RUN mkdir -p /usr/src/cli/
# Binary you get it when you run go build
COPY tenantpool/build/tenantpoolserver /usr/src/app/tenantpoolserver
# CLI binary you get it when you run go build
COPY tenantpool/build/tenantpoolcli /usr/src/cli/tenantpoolcli

EXPOSE 8080

# Change ownership of app directory to tenantpooluser
RUN chown -R tenantpooluser:tenantpoolgroup /usr/src/app
RUN chown -R tenantpooluser:tenantpoolgroup /usr/src/cli

USER tenantpooluser

CMD ["/usr/src/app/tenantpoolserver"]
