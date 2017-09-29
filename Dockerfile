FROM golang:1.9

# Install Glide and Protobuf.
RUN \
  apt-get update && \
  apt-get install -y golang-glide protobuf-compiler libprotobuf-dev

# Install Protobuf plugins.
RUN \
  go get -u github.com/derekparker/delve/cmd/dlv && \
  go get -u github.com/golang/protobuf/protoc-gen-go && \
  go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway && \
  go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger && \
  go get -u github.com/mwitkow/go-proto-validators/protoc-gen-govalidators

# Set startup environment variables.
RUN \
  echo "export POSTGRESQL_ADDRESS=\$(/sbin/ip route|awk '/default/ { print \$3 }'):5432" >> /root/.bashrc

# The work directory will be mapped to the current directory.
WORKDIR /go/src/github.com/tfeng/postgres-grpc-example

SHELL ["/bin/bash", "-i", "-c"]

CMD \
  if [ ! -d vendor ]; then glide install; fi && \
  make && \
  dlv --headless --listen=:2345 --api-version=2 exec /go/bin/pg_server
