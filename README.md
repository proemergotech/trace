# Trace package for the dliver system

Trace packages for the different frameworks we use in dliver.

## Installation

1. Add dependency to go mod
2. Run go build/run/tidy

```bash
go get -u gitlab.com/proemergotech/trace-go v0.3.0
```

## Error logging

The `trace.Error` function will log the error and error fields to the span 
if the error implements the `fielder` interface:

```go
    type fielder interface {
      Fields() []interface{}
    }
```

If the error wrapped other errors and implements the `causer` interface, 
the nested errors and their fields will be logged too.

```go
    type causer interface {
      Cause() error
    }
```

## Documentation

Private repos don't show up on godoc.org so you have to run it locally.

```
godoc -http=":6060"
```

Then open http://localhost:6060/pkg/gitlab.com/proemergotech/trace-go/

## Development

- install go
- check out project to: $GOPATH/src/gitlab.com/proemergotech/trace-go
