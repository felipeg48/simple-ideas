---
type: infographic
title: The 600 ms checkout
subtitle: How a mid-size online store cut its page load by three quarters in six weeks, and what that did to sales
orientation: landscape
aspect: 16:9
layout: grid
theme: studio
author: Web performance team
date: Spring 2026
---

## Median page load · compare · wide
| Before | After |
|---|---|
| 2.4 s | 0.6 s |

## 18% · stat · accent
![](icons/cart.svg)
more checkouts completed in the six weeks after the fix

## Where the 2.4 s went · chart=bar
| Part | ms |
|---|---|
| Images | 920 |
| JavaScript | 610 |
| Server | 340 |
| Fonts | 280 |
| Other | 250 |

## Traffic by device · chart=donut
| Device | % |
|---|---|
| Phones | 64 |
| Desktops | 29 |
| Tablets | 7 |

## How the time came off · steps · wide
1. **Measure** real visitors, not a laptop on office Wi-Fi
2. **Shrink images**: right size, modern formats, lazy below the fold
3. **Split the JavaScript**: checkout loads only what checkout needs
4. **Cache at the edge** and preload the two fonts

## Load time, week by week · chart=line
| Week | Seconds |
|---|---|
| 1 | 2.4 |
| 2 | 2.1 |
| 3 | 1.6 |
| 4 | 1.1 |
| 5 | 0.8 |
| 6 | 0.6 |

## What the team says · quote
> Every hundred milliseconds we saved showed up in the sales graph a week later.
> — The checkout team lead

## Sources · sources
- Real-user monitoring, 1.2 million sessions, weeks 1 to 6
- web.dev, Core Web Vitals (2024)
- HTTP Archive, Web Almanac 2024: Performance

Notes: Our checkout used to take two point four seconds to load for the median visitor. Six weeks later it takes zero point six.
In those same six weeks, eighteen percent more people finished their checkout.
Most of the time went on images, then JavaScript; the server was never the main problem.
And two visitors in three arrive on a phone, where every one of those bytes hurts most.
So we measured real visitors, shrank the images, split the JavaScript and cached at the edge.
Week by week, the line only went one way.
As our team lead put it: every hundred milliseconds showed up in the sales graph a week later.

Points:
- 2.4 s to 0.6 s, median, real users
- +18 % completed checkouts
- Images were the biggest win; the server was fine
- Likely question: did anything else change? (no campaign in those weeks)
