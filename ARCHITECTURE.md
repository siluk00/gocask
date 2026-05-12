# Architecture

> This document captures the design decisions made during implementation,
> the reasoning behind each one, and the alternatives that were explicitly
> rejected. It is updated as the codebase evolves. Sections marked [TODO]
> are planned but not yet implemented.

-----

## Overview

gocask is a log-structured key-value storage engine modeled after the
[Bitcask paper](https://riak.com/assets/bitcask-intro.pdf) (Basho, 2010).
It is designed as a transport-agnostic library, later exposed via HTTP and
gRPC servers and used as the state machine backend for a Raft consensus
implementation.

The core invariant is simple: *every write is an append*. The on-disk log
is never modified in place. An in-memory index (the keydir) maps every live
key to its exact location in the log. A Get performs one index lookup and
one disk seek — no scanning, no binary search.

| Operation | Execution Flow |
| :--- | :--- |
| **Write** | `Key + Value` ➔ Encode ➔ Append to active segment ➔ Update `keydir` |
| **Read** | `Key` ➔ `keydir` lookup ➔ Single `pread` at stored offset ➔ Decode value |
| **Delete**| `Key` ➔ Write tombstone ➔ Remove key from `keydir` |


-----

## On-Disk Format

Every record written to the log has this binary layout:


| CRC32 (4B) | Timestamp (8B) | Key Size (4B) | Val Size (4B) | Key (Var) | Value (Var) |
| :---: | :---: | :---: | :---: | :---: | :---: |
| Offset `0` | Offset `4` | Offset `12` | Offset `16` | Offset `20` | `20 + Key Size` |


Total header size: 20 bytes (HeaderSize constant).

*Why CRC32 first:* corruption is detected before any data is read into
memory. If CRC is at the end, a partial read of a large value would allocate
and copy bytes before discovering they are invalid.

*Why Little-Endian:* consistent behavior on x86 and ARM targets without
conversion overhead. The format is not designed to be portable across
architectures — this is an explicit non-goal.

*Why Timestamp:* two reasons. First, compaction needs to resolve conflicts
when the same key appears in multiple segments — the record with the higher
timestamp wins. Second, TTL support (not yet implemented) will require a
timestamp per record without a format change.

*Tombstone encoding:* deletion reserves bit 31 of the ValSize field as a tombstone flag.
This caps maximum value size at ~2GB (bits 0–30) but avoids ambiguity between
an empty value []byte{} and a deleted key, which a zero-length value
convention would create.

-----

## Design decisions

### Why append-only?

### Why not LSM?

### Deletion via tombstones

### fsync policy

## Known limitations
- Keydir must fit in RAM: ...
- No range queries: ...
- Single writer: ...

## Reading

- Sheehy & Smith - [Bitcask: A Log-Structured Hash Table for Fast Key/Value Data](https://riak.com/assets/bitcask-intro.pdf) (Basho, 2010)
