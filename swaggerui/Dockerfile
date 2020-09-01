FROM registry.access.redhat.com/ubi8/ubi

RUN yum -y install --disableplugin=subscription-manager wget git make \
    && yum --disableplugin=subscription-manager clean all

# Grab go version 1.13
RUN wget https://dl.google.com/go/go1.13.3.linux-amd64.tar.gz

RUN wget https://github.com/mikefarah/yq/releases/download/2.4.1/yq_linux_amd64 && \ 
    chmod +x yq_linux_amd64 && \ 
    mv yq_linux_amd64 /bin/yq
ENV PATH /bin/:$PATH
RUN yq --version

# Install go version 1.13
RUN tar -C /usr/local -xzf go1.13.3.linux-amd64.tar.gz

# Setup go environment variables
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

# Configure application working directories
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

# Change working directory
WORKDIR $GOPATH/src/github.com/ibm/starter-kit-operator

# Install dependencies
ENV GO111MODULE=off
# COPY . ./
# RUN if test -e "go.mod"; then go build ./...; fi

ENV PORT 8080
EXPOSE 8080

RUN pwd && ls && git clone -b web-ui https://github.com/IBM/starter-kit-operator.git .
RUN pwd && yq w ./swaggerui/swagger-template.yaml "host" kubernetes.default.svc.cluster.local > ./swaggerui/swagger.yaml
RUN go get github.com/go-openapi/runtime && go get github.com/jessevdk/go-flags && go get ./swaggerui/cmd/starter-kit-operator-server
RUN go build ./swaggerui/cmd/starter-kit-operator-server
# The certificate and key should be made available via the deployment spec
CMD ["./starter-kit-operator-server", "--tls-certificate", "/kube-api-server.cert.pem", "--tls-key", "/kube-api-server.key.pem"]
