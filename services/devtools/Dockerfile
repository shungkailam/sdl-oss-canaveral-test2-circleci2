# start with alphine
FROM docker.dyn.ntnxdpro.com/ntnx-general-docker/alpine:3.7

# Create devtoolsgroup and devtoolsuser
RUN addgroup -S devtoolsgroup -g 10001 && \
  adduser -D devtoolsuser -u 10001 -G devtoolsgroup
RUN apk add --update-cache ca-certificates

RUN mkdir -p /usr/src/app/
# Binary you get it when you run go build
COPY devtools/build/devtools /usr/src/app/devtools
EXPOSE 8080
EXPOSE 8081

# Change ownership of app directory to devtoolsuser
RUN chown -R devtoolsuser:devtoolsgroup /usr/src/app

USER devtoolsuser

CMD ["/usr/src/app/devtools"]