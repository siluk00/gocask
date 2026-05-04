# Architecture

## Overview


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
