PROTO_OBJECTS = auth/auth.pb.go models/user.auth.pb.go models/user.pb.go models/user.rest.pb.go models/user.validator.pb.go
PROTOC_INCLUDES = -Ivendor -Ivendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I$(GOPATH)/src

all: install

install: \
	$(GOPATH)/bin/protoc-gen-goauth \
	$(GOPATH)/bin/protoc-gen-gorest \
	$(PROTO_OBJECTS) \
	$(GOPATH)/bin/pg_client \
	$(GOPATH)/bin/pg_server

%.auth.pb.go: %.proto $(GOPATH)/bin/protoc-gen-goauth
	protoc $(PROTOC_INCLUDES) --proto_path=. --goauth_out=. $<

%.pb.go: %.proto
	protoc $(PROTOC_INCLUDES) --proto_path=. --go_out=plugins=grpc,Mgoogle/protobuf/descriptor.proto=github.com/golang/protobuf/protoc-gen-go/descriptor:. $<

%.rest.pb.go: %.proto $(GOPATH)/bin/protoc-gen-gorest
	protoc $(PROTOC_INCLUDES) --proto_path=. --gorest_out=. $<

%.validator.pb.go: %.proto
	protoc $(PROTOC_INCLUDES) --proto_path=. --govalidators_out=. $<

$(GOPATH)/bin/protoc-gen-goauth: auth/protoc-gen-goauth/*.go auth/auth.pb.go
	go install github.com/tfeng/postgres-grpc-example/auth/protoc-gen-goauth

$(GOPATH)/bin/protoc-gen-gorest: rest/protoc-gen-gorest/*.go
	go install github.com/tfeng/postgres-grpc-example/rest/protoc-gen-gorest

$(GOPATH)/bin/pg_client: pg_client/*.go $(PROTO_OBJECTS)
	go install github.com/tfeng/postgres-grpc-example/pg_client

$(GOPATH)/bin/pg_server: pg_server/*.go auth/auth.go $(PROTO_OBJECTS)
	go install github.com/tfeng/postgres-grpc-example/pg_server

clean: uninstall

uninstall:
	rm -rf $(GOPATH)/bin/protoc-gen-goauth $(GOPATH)/bin/protoc-gen-gorest $(GOPATH)/bin/pg_client $(GOPATH)/bin/pg_server $(PROTO_OBJECTS) *~

docker-build:
	docker build --tag=postgres-grpc-example .

docker-run:
	docker run --rm -v "${CURDIR}":/go/src/github.com/tfeng/postgres-grpc-example -p 8080:8080 -p 9090:9090 -it postgres-grpc-example
