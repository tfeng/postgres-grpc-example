PROTO_OBJECTS = auth/auth.pb.go models/user.auth.pb.go models/user.pb.go models/user.rest.pb.go models/user.validator.pb.go

all: install

install: \
	$(GOPATH)/bin/protoc-gen-goauth \
	$(GOPATH)/bin/protoc-gen-gorest \
	$(PROTO_OBJECTS) \
	$(GOPATH)/bin/pg_client \
	$(GOPATH)/bin/pg_server

%.auth.pb.go: %.proto $(GOPATH)/bin/protoc-gen-goauth
	protoc -I$(GOPATH)/src \
	  -I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	  --proto_path=. \
	  --goauth_out=. \
	  $<

%.pb.go: %.proto
	protoc -I$(GOPATH)/src \
	  -I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	  --proto_path=. \
	  --go_out=plugins=grpc:. \
	  $<

%.rest.pb.go: %.proto $(GOPATH)/bin/protoc-gen-gorest
	protoc -I$(GOPATH)/src \
	  -I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	  --proto_path=. \
	  --gorest_out=. \
	  $<

%.validator.pb.go: %.proto
	protoc -I$(GOPATH)/src \
	  -I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	  --proto_path=. \
	  --govalidators_out=. \
	  $<

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
	rm -f $(GOPATH)/bin/protoc-gen-goauth $(GOPATH)/bin/protoc-gen-gorest $(GOPATH)/bin/pg_client $(GOPATH)/bin/pg_server $(PROTO_OBJECTS)
