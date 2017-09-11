PROTO_FILES = models/user.proto
PROTO_OBJECTS = $(PROTO_FILES:%.proto=%.pb.go) $(PROTO_FILES:%.proto=%.pb.gw.go) $(PROTO_FILES:%.proto=%.validator.pb.go)

all: install

install: \
	$(PROTO_OBJECTS) \
	$(GOPATH)/bin/pg_client \
	$(GOPATH)/bin/pg_gateway \
	$(GOPATH)/bin/pg_server

%.pb.go: %.proto
	protoc -I$(GOPATH)/src \
	  -I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	  --proto_path=. \
	  --go_out=plugins=grpc:. \
	  $<

%.pb.gw.go: %.proto
	protoc -I$(GOPATH)/src \
	  -I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	  --proto_path=. \
	  --grpc-gateway_out=. \
	  $<

%.validator.pb.go: %.proto
	protoc -I$(GOPATH)/src \
	  -I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	  --proto_path=. \
	  --govalidators_out=. \
	  $<

$(GOPATH)/bin/pg_client: pg_client/*.go $(PROTO_OBJECTS)
	go install github.com/tfeng/postgres-grpc-example/pg_client

$(GOPATH)/bin/pg_gateway: pg_gateway/*.go $(PROTO_OBJECTS)
	go install github.com/tfeng/postgres-grpc-example/pg_gateway

$(GOPATH)/bin/pg_server: pg_server/*.go $(PROTO_OBJECTS)
	go install github.com/tfeng/postgres-grpc-example/pg_server

clean: uninstall

uninstall:
	rm -f $(PROTO_OBJECTS) $(GOPATH)/bin/pg_client $(GOPATH)/bin/pg_gateway $(GOPATH)/bin/pg_server
