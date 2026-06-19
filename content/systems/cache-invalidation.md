---
title: Cache Invalidation
slug: cache-invalidation
category: Systems
difficulty: intermediate
reading_time: 8 min read
tags:
  - systems
  - caching
  - reliability
description: Use freshness, ownership, and failure modes to reason about cache invalidation.
---

## Start with ownership

Every cache needs a source of truth. Before choosing time-to-live values or invalidation events, identify the writer that owns the data and the readers that can tolerate stale responses.

## Freshness strategies

Time-based freshness is easy to operate and works well for data that changes slowly. Event-based invalidation gives tighter correctness, but only when the event stream is reliable and replayable.

Many production systems combine both:

```text
read from cache
if miss or expired:
  fetch source of truth
  write cache with TTL
  return value
```

## Failure modes

- Stampedes happen when many readers miss at once.
- Silent stale data appears when invalidation events are dropped.
- Hot keys can overload one cache partition.
- Negative caching can hide newly-created data if the TTL is too long.

Good cache design names the acceptable stale window and makes that tradeoff visible.
