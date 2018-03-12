# Trace package for the dliver system

Trace packages for the different frameworks we use in dliver.

## Installation

1. Add dependency to dep
2. Run dep ensure

#### Gopkg.toml

```toml
[[constraint]]
  name = "gitlab.com/proemergotech/trace-go"
  source = "git@gitlab.com:proemergotech/trace-go.git"
  version = "0.1.0"
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
- check out project to: $GOPATH/src/gitlab.com/proemergotech/log-go
- install dep
- run dep ensure
