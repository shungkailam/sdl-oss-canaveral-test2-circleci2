# start with alphine
FROM docker.dyn.ntnxdpro.com/ntnx-general-docker/alpine:3.7

RUN apk add --update-cache ca-certificates

# Create accountgroup and accountuser
RUN addgroup -S accountgroup -g 10001 && \
  adduser -D accountuser -u 10001 -G accountgroup

RUN mkdir -p /usr/src/app/
#Binary you get it when you run go build
COPY account/build/accountserver /usr/src/app/accountserver
EXPOSE 8080

# Change ownership of app directory to accountuser
RUN chown -R accountuser:accountgroup /usr/src/app

USER accountuser

CMD ["/usr/src/app/accountserver"]
