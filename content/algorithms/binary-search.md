---
title: Binary Search
slug: binary-search
category: Algorithms
difficulty: beginner
reading_time: 7 min read
tags:
  - algorithms
  - arrays
  - search
description: Build the invariant behind binary search before memorizing the loop.
---

## The core idea

Binary search works when your data or answer space is ordered. Each check discards one side of the remaining range without losing the answer.

The useful mental model is not "check the middle." It is "preserve the invariant that the answer is still inside the active range."

```go
func BinarySearch(xs []int, target int) int {
	left, right := 0, len(xs)-1
	for left <= right {
		mid := left + (right-left)/2
		switch {
		case xs[mid] == target:
			return mid
		case xs[mid] < target:
			left = mid + 1
		default:
			right = mid - 1
		}
	}
	return -1
}
```

## Common mistakes

- Updating `left` or `right` without excluding `mid`.
- Mixing inclusive and half-open ranges in the same loop.
- Calculating `mid` in a way that can overflow in languages with fixed integer widths.
