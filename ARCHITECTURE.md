# Architecture

## Overview

Bitcask is a high-performance, log-structured key-value store inspired by the storage engine used in Riak. It relies on an append-only file format where all writes are sequentially added to an active data file. To maintain fast read performance, it employs an in-memory data structure called a 'keydir'—a hash map that stores the latest file location and offset for every key. This design ensures that every read requires at most one disk seek and every write is a simple append, making it exceptionally efficient for write-heavy workloads while maintaining predictable read latency.

gocask is a Go implementation of the Bitcask design, focusing on simplicity, safety, and modern concurrency. The system manages data through a series of immutable segments, automatically rotating the active segment when it reaches a configurable size limit. It leverages Go's concurrency primitives and atomic operations to allow safe, simultaneous access from multiple goroutines, ensuring that reads never block writes. While currently in a functional prototype stage, gocask supports the core Bitcask operations—Put, Get, and Delete (via tombstones)—and is designed to be easily extensible for future features like hint files and automatic merging.

## On-disk format


## The keydir


## Design decisions

### Why append-only?

### Why not LSM?

### Deletion via tombstones

### fsync policy

## Known limitations
- Keydir must fit in RAM: ...
- No range queries: ...
- Single writer: ...
