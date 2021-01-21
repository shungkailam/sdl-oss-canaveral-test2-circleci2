FROM docker.dyn.ntnxdpro.com/ntnx-general-docker/node:8.9-alpine
RUN apk add --update-cache ca-certificates
RUN mkdir -p /usr/client-certs
RUN mkdir -p /usr/src/app/ui/dist/docs
RUN mkdir -p /usr/src/app/api/dist
RUN mkdir -p /usr/common
RUN apk --update add openjdk8-jre openssh-keygen
RUN mkdir /helmApp
RUN chmod a+rw /helmApp
RUN mkdir -p /usr/src/app/cli
WORKDIR /usr/src/app
COPY ./nodejs/api/package.json ./api/
COPY ./nodejs/api/yarn.lock ./api/
RUN cd api && yarn --prod
COPY ./common/kubeval /usr/common/kubeval
COPY ./nodejs/api/src/swagger-ui ./ui/dist/docs/
COPY ./nodejs/api/dist ./api/dist/
COPY ./cloudmgmt/build/ui/dist ./ui/dist/
COPY ./cloudmgmt/build/ui-version.txt ./ui/dist/ui-version.txt
COPY ./cloudmgmt/build/robots.txt ./ui/dist/robots.txt
#from swaggerspec
COPY ./cloudmgmt/swagger.json ./ui/dist/xi_iot_api.json
COPY ./cloudmgmt/swagger_full.json ./ui/dist/xi_iot_api_full.json
COPY ./cloudmgmt/swagger.json ./ui/dist/kps_api.json
COPY ./cloudmgmt/swagger_full.json ./ui/dist/kps_api_full.json
COPY ./cloudmgmt/sureroute-test-object.html ./ui/dist/sureroute-test-object.html

# Note: Latest version of helm may be found at:
# https://github.com/kubernetes/helm/releases
ENV HELM_VERSION="v2.14.3"
RUN apk add --no-cache ca-certificates bash git curl openssl \
    && wget -q https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz -O - | tar -xzO linux-amd64/helm > /usr/local/bin/helm \
    && chmod +x /usr/local/bin/helm
#Binary you get it when you run go build
COPY ./cloudmgmt/build/cloudmgmt ./cloudmgmt

# Copy the Service Class CLI in tool
COPY ./tool/build/serviceclassupserter ./cli/serviceclassupserter
RUN chmod +x ./cli/serviceclassupserter

ENV CONTENTDIR=/usr/src/app/ui/dist
EXPOSE 8080
CMD ["/usr/src/app/cloudmgmt"]
