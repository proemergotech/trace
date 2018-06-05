# Release Notes

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
