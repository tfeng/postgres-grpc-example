# postgres-grpc-example

## Install

Dependencies are managed with [Glide](https://github.com/Masterminds/glide). Therefore, make sure it is installed and run this to pull the dependencies.

```$bash
$ glide install
```

In addition, the following commands make sure global dependencies are installed.

```bash
$ go get -u github.com/golang/protobuf/protoc-gen-go
$ go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
$ go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
$ go get -u github.com/mwitkow/go-proto-validators/protoc-gen-govalidators
```

Build the project with [GNU Make](https://www.gnu.org/software/make/).

```$bash
$ make
```

## Start Postgres

This demo uses [PostgreSQL](https://www.postgresql.org/) to store user data. As a prerequisite, the database needs to be
started in a [Docker](https://www.docker.com/) container.

```$bash
$ docker run --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=password --rm -d postgres
```

To connect to the database, the program uses [go-pg](https://github.com/go-pg/pg).

## Start server

In one terminal, run this to start the [GRPC](https://grpc.io/) server. It listens to http://localhost:9090 through GRPC
protocol, and it also exposes a Rest API at http://localhost:8080.

```$bash
$ pg_server
```

## Make Rest requests

### Get client access token

The following command obtains an [OAuth2 token](https://www.oauth.com/oauth2-servers/access-tokens/access-token-response/).
This token authorizes a client with id "client".

```$bash
$ curl -X POST -H 'Content-Type: application/json' -d '{"client_id": "client", "client_secret": "password", "grant_type": "client_credentials"}' 'localhost:8080/oauth/tokens'
```

One can save the access token into the `CLIENT_TOKEN` environment variable.

```$bash
$ CLIENT_TOKEN=$(curl -s -X POST -H 'Content-Type: application/json' -d '{"client_id": "client", "client_secret": "password", "grant_type": "client_credentials"}' 'localhost:8080/oauth/tokens' | jq -r '.access_token')
```

### Create a user

The following command uses the client OAuth2 token to create a user.

```$bash
$ curl -X POST -H 'Content-Type: application/json' -H "authorization: bearer $CLIENT_TOKEN" -d '{"username": "tfeng", "password": "password"}' localhost:8080/v1/users/create
```

### Authenticate

The following command uses the client OAuth2 token and the username and password to obtain another OAuth2 token that
authorizes operations on a specific user.

```$bash
$ USER_TOKEN=$(curl -s -X POST -H "Authorization: Bearer $CLIENT_TOKEN" -H 'Content-Type: application/json' -d '{"username": "tfeng", "password": "password", "grant_type": "password"}' 'localhost:8080/oauth/tokens' | jq -r '.access_token')
```

The returned JSON is an [OAuth2 token](https://www.oauth.com/oauth2-servers/access-tokens/access-token-response/) token.

### Get the current user

The following command fetches the profile information of the user.

```$bash
$ curl -H 'Content-Type: application/json' -H "authorization: bearer $USER_TOKEN" localhost:8080/v1/users/get
```

## Make GRPC requests

A GRPC client can directly make requests to the server, without going through the gateway.

```$bash
$ pg_client
```

This will execute the same steps as above, i.e., creating a user, logging in as that user, and getting that user's
information.
