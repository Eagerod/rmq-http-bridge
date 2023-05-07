# RabbitMQ HTTP Bridge

A two way HTTP bridge for RabbitMQ.

## Modes of operation

**Init:** Creates the RabbitMQ exchange architecture needed to implement delays.

**Producer:** Starts an HTTP server that allows producers to send in endpoint/task definitions to RMQ.

**Consumer:** Consumes task definitions from RMQ, and calls the HTTP endpoints with the provided headers + payload.

## Development

Hope uses Go 1.19, and requires staticcheck to be installed to run its linting/validation recipes:

```
go install honnef.co/go/tools/cmd/staticcheck@v0.4.2
```
