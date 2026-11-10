---
title: Marta Lindqvist
description: Staff engineer, order platform. I keep forty thousand orders a day moving from checkout to the dock, and I help teams run what they build.
---

**marta.lindqvist@northwind.example** · +46 31 555 0142 ·
mlindqvist.northwind.example · Gothenburg, Sweden

<!-- badge: Open to staff and principal roles from March 2027 · ok -->

## Experience

### Staff engineer, order platform · Northwind Logistics · 2022 to now

- Led the move of checkout, payment and stock reservation from one database
  to three services joined by a queue; p95 checkout fell from 1.9 s to 410 ms.
- Wrote the retry and idempotency rules every service now follows; duplicate
  charges went from about forty a month to none in eighteen months.
- Run the incident review: a written review within five days, and one change
  to the training lab after each.

### Senior backend engineer · Northwind Logistics · 2018 to 2022

- Built the courier hand-off service: eleven carriers behind one API, and a
  fallback carrier chosen in under a second when one stops answering.
- Mentored five engineers; three now lead teams of their own.

### Backend engineer · Kattegat Freight · 2014 to 2018

- Moved container booking from nightly batches to events; customers saw
  their containers move within a minute instead of the next morning.

## Skills

| Area | What I use |
|---|---|
| Languages | Go, Java, SQL, Python |
| Systems | Message queues, PostgreSQL, distributed transactions, Kubernetes |
| Practice | Incident review, capacity planning, on-call design, technical writing |
| Spoken | Swedish (native), English (fluent), Portuguese (working) |

## Education

### MSc, Computer Science · West Coast Institute of Technology · 2014

Thesis: *Back-pressure in message pipelines with bursty arrivals.*

### BSc, Software Engineering · West Coast Institute of Technology · 2012

## Talks and writing

- *Retries without the stampede*, Northwind Engineering, 2026
- *The order journey*, Gothenburg Go meetup, 2025
