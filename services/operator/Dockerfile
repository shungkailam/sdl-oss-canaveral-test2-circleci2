# start with alphine
FROM docker.dyn.ntnxdpro.com/ntnx-general-docker/alpine:3.7

# Create operatorgroup and operatoruser
RUN addgroup -S operatorgroup -g 10001 && \
  adduser -D operatoruser -u 10001 -G operatorgroup

RUN apk add --update-cache ca-certificates

RUN mkdir -p /usr/src/app/
#Binary you get it when you run go build
COPY operator/build/operatorserver /usr/src/app/operatorserver

# Change ownership of app directory to operatoruser
RUN chown -R operatoruser:operatorgroup /usr/src/app

USER operatoruser

CMD ["/usr/src/app/operatorserver"]
