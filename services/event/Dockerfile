# start with alphine
FROM docker.dyn.ntnxdpro.com/ntnx-general-docker/alpine:3.7

# Create eventgroup and eventuser
RUN addgroup -S eventgroup -g 10001 && \
  adduser -D eventuser -u 10001 -G eventgroup
RUN apk add --update-cache ca-certificates

RUN mkdir -p /usr/src/app/
# Binary you get it when you run go build
COPY event/build/eventserver /usr/src/app/eventserver
EXPOSE 8080

# Change ownership of app directory to eventuser
RUN chown -R eventuser:eventgroup /usr/src/app

USER eventuser

CMD ["/usr/src/app/eventserver"]
