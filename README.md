# postgres-grpc-example

## Install

```$bash
$ make
```

## Start Postgres

This demo uses [PostgreSQL](https://www.postgresql.org/) to store user data. As a prerequisite, the database needs to be started in a [Docker](https://www.docker.com/) container.

```$bash
$ docker run --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=password --rm -d postgres
```

To connect to the database, the program uses [go-pg](https://github.com/go-pg/pg).

## Start server

In one terminal, run this to start the [grpc](https://grpc.io/) server. It listens to http://localhost:9090.
```$bash
$ pg_server
```

In another terminal, run this to start the [grpc gateway](https://github.com/grpc-ecosystem/grpc-gateway). It listens to http://localhost:8080, and proxies requests to the grpc server.
```$bash
$ pg_gateway
```

## Make requests through the gateway

### Add a user

```$bash
$ curl -X POST -d '{"name": "Thomas", "password": "password", "role": "admin"}' localhost:8080/v1/user/create
```

### Login

```$bash
$ TOKEN=$(curl -X POST -d '{"id": 1, "password": "password"}' localhost:8080/v1/user/login | jq -r '.token')
```

The returned JSON contains only a [JWT](https://jwt.io/) token. It has information of the logged in user, encrypted by a random RSA key with the [crypto](https://github.com/golang/crypto) package.

### Get the current user

```$bash
$ curl -H "authorization: bearer $TOKEN" localhost:8080/v1/user/get
```

The token obtained in the previous step is passed to the server with the HTTP "authorization" header. Because the token already has all the information needed for the authorization, the server need not read from the database for authorization.

## Make requests to the server

A grpc client can directly make requests to the server, without going through the gateway.

```$bash
$ pg_client
```

This will execute the same steps as above, i.e., creating a user, logging in as that user, and getting that user's information.
