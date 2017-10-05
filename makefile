PROTO_OBJECTS = auth/auth.auth.pb.go auth/auth.pb.go auth/auth.rest.pb.go models/user/user.auth.pb.go models/user/user.pb.go models/user/user.rest.pb.go models/user/user.validator.pb.go
PROTOC_INCLUDES = -Ivendor -Ivendor/github.com/golang/protobuf -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$(GOPATH)/src

all: install

install: \
	$(GOPATH)/bin/protoc-gen-goauth \
	$(GOPATH)/bin/protoc-gen-gorest \
	$(PROTO_OBJECTS) \
	$(GOPATH)/bin/pg_client \
	$(GOPATH)/bin/pg_server

auth/auth.auth.pb.go: auth/auth.proto $(GOPATH)/bin/protoc-gen-goauth
	protoc $(PROTOC_INCLUDES) --proto_path=. --goauth_out=auth_package:. $<

%.auth.pb.go: %.proto $(GOPATH)/bin/protoc-gen-goauth
	protoc $(PROTOC_INCLUDES) --proto_path=. --goauth_out=. $<

%.pb.go: %.proto
	protoc $(PROTOC_INCLUDES) --proto_path=. --go_out=plugins=grpc,Mgoogle/protobuf/descriptor.proto=github.com/golang/protobuf/protoc-gen-go/descriptor:. $<

%.rest.pb.go: %.proto $(GOPATH)/bin/protoc-gen-gorest
	protoc $(PROTOC_INCLUDES) --proto_path=. --gorest_out=. $<

%.validator.pb.go: %.proto
	protoc $(PROTOC_INCLUDES) --proto_path=. --govalidators_out=. $<

$(GOPATH)/bin/protoc-gen-goauth: auth/protoc-gen-goauth/*.go auth/auth.pb.go auth/auth.rest.pb.go
	go install github.com/tfeng/postgres-grpc-example/auth/protoc-gen-goauth

$(GOPATH)/bin/protoc-gen-gorest: rest/protoc-gen-gorest/*.go
	go install github.com/tfeng/postgres-grpc-example/rest/protoc-gen-gorest

$(GOPATH)/bin/pg_client: pg_client/*.go $(PROTO_OBJECTS)
	go install github.com/tfeng/postgres-grpc-example/pg_client

$(GOPATH)/bin/pg_server: pg_server/*.go auth/*.go config/*.go models/user/*.go rest/*.go $(PROTO_OBJECTS)
	go install github.com/tfeng/postgres-grpc-example/pg_server

clean: uninstall

uninstall:
	rm -rf $(GOPATH)/bin/protoc-gen-goauth $(GOPATH)/bin/protoc-gen-gorest $(GOPATH)/bin/pg_client $(GOPATH)/bin/pg_server $(PROTO_OBJECTS) *~

docker-build:
	docker build --tag=postgres-grpc-example .

docker-run:
	docker run --privileged --rm -v "${CURDIR}":/go/src/github.com/tfeng/postgres-grpc-example -p 2345:2345 -p 8080:8080 -p 9090:9090 -it postgres-grpc-example bash -i -c "if [ ! -d vendor ]; then glide install; fi && make && /go/bin/pg_server"

docker-debug:
	docker run --privileged --rm -v "${CURDIR}":/go/src/github.com/tfeng/postgres-grpc-example -p 2345:2345 -p 8080:8080 -p 9090:9090 -it postgres-grpc-example bash -i -c "if [ ! -d vendor ]; then glide install; fi && make && dlv --headless --listen=:2345 --api-version=2 exec /go/bin/pg_server"
