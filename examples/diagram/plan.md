---
type: diagram
title: Parcel lockers, the plan
description: Three weeks from design to launch, and where we stand
theme: studio
---

```gantt start=2026-12-01
section Build
Design · 5d
Locker API · 8d · after Design
Web map · 6d · after Design
section Ship
Tests · 4d · after Locker API, Web map
Launch · milestone · after Tests

today 2026-12-04 · Design 80%
today 2026-12-10 · Design done, Locker API 60%, Web map done
today 2026-12-16 · Locker API 90%, Tests 20%
```
