# Release Notes

## v2.0.0 / 2020-02-05
- update geb client to v2
- release version v2.0.0


## v1.0.0 / 2020-01-07
- release version v1.0.0
- add verify script and ci job 

## v0.3.0 / 2019-12-12
- use go modules
- update to echo v4
- remove gin

## v0.2.4 / 2019-11-05
- fix gentleman trace message

## v0.2.3 / 2019-07-19
- shortened echo, gin & gentleman trace spans for easier usage in operations filter 

## v0.2.2 / 2019-06-12
- added trace handling options to gentleman middleware

## v0.2.1 / 2018-09-29
- fix index out of range error in gin trace

## 0.2.0 / 2018-09-24
- throw error if the correlation id missing from header or context

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
