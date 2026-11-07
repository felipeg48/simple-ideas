---
type: article
title: {{title}}
description: One change to the code, step by step.
date: {{date}}
---

What the code does today, in one sentence.

```go id=change window=mac title="send.go" numbers step="Today"
func send(order Order) error {
    return client.Post(url, order)
}
```

```go id=change step="With a retry"
func send(order Order) error {
    var last error
    for try := 1; try <= 3; try++ {
        if last = client.Post(url, order); last == nil {
            return nil
        }
    }
    return last
}
```
<!-- callout: 3 Three tries, then give up -->

Why the change matters, in one sentence.
