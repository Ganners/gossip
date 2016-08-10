Gossip Microservice Framework
=============================

This is a proof of concept for a purely gossip based microservice framework
built in Go. It is missing a lot of pieces and optimisations but it roughly
achieves what it sets out to do.

This reduces the need for shared infrastructure to handle reliable
communication, a microservice built around gossiping is therefore purely
`reactive` and very resilient to failure.

There are a large number of fairly important TODO statements in this, for the
purpose of the Yoti test this is set out to solve this doesn't need to be 100%
complete. The underlying architecture and API can be criticised heavily before
proceeding with the finer implementation details.

There are various potential implementations for particular parts which would be
good to compose in and benchmark in future. The literature on this is fairly
ubiquitous although it hasn't gained much ground as the core microservice
transport protocol The literature on this is fairly ubiquitous although it
hasn't gained much ground as the core microservice transport protocol.

For testing I've built a couple of programs in the example folder. To start
simply launch them in order in a couple of terminals.

> go run example/server1.go

> go run example/server2.go

Server 2 makes login requests to server 1, server 1 writes responses to these.
Both servers maintain a view of their world (excluding themselves). I've made
the node list pretty print so it can be easily visualised.
