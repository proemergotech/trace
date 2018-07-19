# Release Notes

## 0.1.8 / 2018-07-19
- fixed echo tracer not adding context to request

## 0.1.7 / 2018-06-21
- added Echo web framework tracer.

## 0.1.6 / 2018-06-05
- fixed NewCorrelationID

## 0.1.5 / 2018-06-05
- added NewCorrelation and NewCorrelationID methods 

## 0.1.4 / 2018-04-19
- fixed sharing a span reference between all gentleman requests

## 0.1.3 / 2018-04-03
- removed version contraint from geb-client dependency

## 0.1.2 / 2018-03-28
- do not mark the span as failed when trace not found
- add a "start.ignored" tag, when a Start option provided, but the incoming request/event has trace information too

## 0.1.1 / 2018-03-19
- added option to start trace from gin and gebOnEvent middlewares

## 0.1.0 / 2018-03-12
- project created
