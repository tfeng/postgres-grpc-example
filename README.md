# postgres-grpc-example

## Add a user

```$bash
$ curl -X POST -d '{"name": "Thomas"}' localhost:8080/v1/user/add
```

## Get a user

```$bash
$ curl -X POST -d '{"id": 1}' localhost:8080/v1/user/get
```
