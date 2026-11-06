---
title: AceMQ Pro theme
theme: acemq-pro
footer: Source: Northwind Pay incident reports, Aug–Sep 2026
---

<!-- footer: off -->

# Payments messaging that ==survives a lost zone.==

<div class="floaters"><div class="fl">AMQP</div><div class="fl">MQTT</div><div class="fl">K8s</div></div>

<!-- logo: × Northwind Pay · alt=AceMQ -->

<!-- badge: Prepared for Northwind Pay · November 2026 -->

<p class="cover-sub">One broker, run by the people who build it.</p>

<p class="lead">Quorum queues by default, zone-aware placement, and a migration that keeps every message.</p>

---

<!-- eyebrow: August – September 2026 -->

## Six numbers from ==your own tickets.==

<div class="heard">
<div class="hc"><span class="k">Peak rate</span><span class="big">42<em>k/s</em></span><p>Card authorisations at the Friday peak.</p></div>
<div class="hc"><span class="k">Incidents</span><span class="big">4</span><p>Broker-related outages in eight weeks.</p></div>
<div class="hc hi"><span class="k">Worst outage</span><span class="big">47<em>min</em></span><p>A lost zone took the only queue leader with it.</p></div>
<div class="hc"><span class="k">Clusters</span><span class="big">3</span><p>One per region, managed by hand.</p></div>
<div class="hc"><span class="k">Messages lost</span><span class="big">0</span><p>Every outage was downtime, not data loss.</p></div>
<div class="hc"><span class="k">On-call pages</span><span class="big">31</span><p>Most of them for disk alarms.</p></div>
</div>

---

<!-- eyebrow: Incidents -->

## Four outages, ==one root cause.==

<div class="inc">
<div class="ic"><span class="when">Aug 04</span><h4>Zone b lost</h4><p>Leaders on b stopped; 47 min to recover.</p></div>
<div class="ic"><span class="when">Aug 21</span><h4>Disk alarm</h4><p>Publishers blocked for 12 min.</p></div>
<div class="ic"><span class="when">Sep 02</span><h4>Partition</h4><p>Pause-minority stopped two nodes.</p></div>
<div class="ic"><span class="when">Sep 18</span><h4>Upgrade</h4><p>A rolling restart drained a classic mirror.</p></div>
</div>
<p class="verdict">Every one of them is a single queue leader in one zone. <b>Quorum placement removes it.</b></p>

---

<!-- eyebrow: The core problem -->

## A zone is a node, ==until it fails.==

<div class="split2">
<ul class="blist">
<li><b>Leaders cluster</b> on whichever node started first.</li>
<li class="crit"><b>A lost zone</b> takes every leader it holds.</li>
<li class="ok"><b>Quorum queues</b> elect a new leader in seconds.</li>
</ul>
<div class="zones">
<div class="zg"><span class="zl">Zone a</span><div class="zc ok">node-1 · 3 leaders</div><div class="zc">node-2</div></div>
<div class="zg"><span class="zl">Zone b</span><div class="zc down">node-3 · 9 leaders</div><div class="zc down">node-4</div></div>
<div class="zg"><span class="zl">Zone c</span><div class="zc hot">node-5 · new leaders</div><div class="zc">node-6</div></div>
</div>
</div>

---

<!-- eyebrow: Options -->

## Four ways forward, ==one we recommend.==

<div class="opts">
<div class="opt"><span class="ol">A</span><span class="tag">Today</span><h4>Keep as is</h4><p>Hand-run clusters, classic mirrors.</p><dl><div><dt>Lost zone</dt><dd>Outage</dd></div><div><dt>Effort</dt><dd>None</dd></div></dl></div>
<div class="opt"><span class="ol">B</span><span class="tag">Cloud</span><h4>Managed MQ</h4><p>Same broker, someone else's pager.</p><dl><div><dt>Lost zone</dt><dd>Outage</dd></div><div><dt>Effort</dt><dd>Medium</dd></div></dl></div>
<div class="opt hi"><span class="ol">C</span><span class="tag">Recommended</span><h4>AceMQ Broker</h4><p>Quorum by default, zone-aware.</p><dl><div><dt>Lost zone</dt><dd>Alert</dd></div><div><dt>Effort</dt><dd>Medium</dd></div></dl></div>
<div class="opt"><span class="ol">D</span><span class="tag">Rewrite</span><h4>Streaming</h4><p>A new platform and new clients.</p><dl><div><dt>Lost zone</dt><dd>Alert</dd></div><div><dt>Effort</dt><dd>High</dd></div></dl></div>
</div>

---

<!-- eyebrow: Side by side -->

## Same workload, ==fewer surprises.==

<table class="tbl">
<thead><tr><th></th><th>Today</th><th>Managed MQ</th><th class="us">AceMQ</th></tr></thead>
<tbody>
<tr><td>Lost zone</td><td><span class="n">Outage</span></td><td><span class="n">Outage</span></td><td class="us"><span class="y">Leader moves</span></td></tr>
<tr><td>Disk alarm</td><td><span class="n">Blocks</span></td><td><span class="p">Blocks</span></td><td class="us"><span class="y">Spills</span></td></tr>
<tr><td>Upgrades</td><td><span class="p">Manual</span></td><td><span class="y">Managed</span></td><td class="us"><span class="y">Managed</span></td></tr>
</tbody>
</table>

---

<!-- eyebrow: What changes -->

## The numbers that ==move first.==

<div class="facts">
<div class="fr"><span class="fn">47<em>min</em></span><div><h4>to under 10 seconds</h4><p>A lost zone becomes a leader election, not an outage.</p></div></div>
<div class="fr"><span class="fn">31</span><div><h4>pages to a handful</h4><p>Disk alarms spill to the next node instead of blocking publishers.</p></div></div>
<div class="fr"><span class="fn">3</span><div><h4>clusters, one control plane</h4><p>All regions from one console and one runbook.</p></div></div>
</div>

---

<!-- eyebrow: AceMQ Broker -->

## Built for the day ==a zone goes dark.==

<div class="feat">
<div class="fc"><span class="icon-chip"><svg viewBox="0 0 24 24"><path d="M12 3l8 4v5c0 5-3.5 8-8 9-4.5-1-8-4-8-9V7z"/></svg></span><h4>Quorum by default</h4><p>Every queue replicated across three zones.</p></div>
<div class="fc"><span class="icon-chip icon-chip--teal"><svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="8"/><path d="M12 8v4l3 2"/></svg></span><h4>Fast failover</h4><p>New leaders elected in seconds.</p></div>
<div class="fc"><span class="icon-chip"><svg viewBox="0 0 24 24"><path d="M4 7h16M4 12h16M4 17h10"/></svg></span><h4>Disk spill</h4><p>Back-pressure without blocking publishers.</p></div>
<div class="fc"><span class="icon-chip icon-chip--ink"><svg viewBox="0 0 24 24"><path d="M5 12l4 4 10-10"/></svg></span><h4>Drop-in AMQP</h4><p>Your clients connect unchanged.</p></div>
<div class="fc"><span class="icon-chip"><svg viewBox="0 0 24 24"><rect x="4" y="4" width="16" height="16" rx="3"/><path d="M9 12h6"/></svg></span><h4>One console</h4><p>Every cluster, every region.</p></div>
<div class="fc"><span class="icon-chip icon-chip--teal"><svg viewBox="0 0 24 24"><path d="M4 18l5-6 4 3 7-9"/></svg></span><h4>Live metrics</h4><p>Queue depth and lag per tenant.</p></div>
</div>

---

<!-- eyebrow: Benchmark · 3 nodes, 1 KB messages -->
<!-- footer: Source: AceMQ lab, c7i.2xlarge, persistent messages -->

## Twice the rate, ==a third of the lag.==

<div class="bench">
<div class="brow us" style="--w:92%"><span class="bl">AceMQ Broker</span><span class="bar"></span><span class="bv">84k/s</span></div>
<div class="brow" style="--w:46%"><span class="bl">Classic mirrors</span><span class="bar"></span><span class="bv">41k/s</span></div>
<div class="brow" style="--w:38%"><span class="bl">Managed MQ</span><span class="bar"></span><span class="bv">35k/s</span></div>
</div>
<div class="bks">
<div class="bk hi"><span class="big">8<em>ms</em></span><p>p99 publish latency</p></div>
<div class="bk"><span class="big">6<em>s</em></span><p>failover after a lost zone</p></div>
<div class="bk"><span class="big">0</span><p>messages lost in 200 chaos runs</p></div>
</div>

---

<!-- eyebrow: Migration -->

## Six steps, ==no message left behind.==

<div class="flowline">
<div class="fs"><h4>Inventory</h4><p>Queues, policies, clients.</p></div>
<div class="fs"><h4>Shadow</h4><p>AceMQ beside today's cluster.</p></div>
<div class="fs hot"><h4>Shovel</h4><p>Federate the busiest queues.</p></div>
<div class="fs"><h4>Cut over</h4><p>Consumers first, then publishers.</p></div>
<div class="fs"><h4>Drain</h4><p>Empty the old queues.</p></div>
<div class="fs"><h4>Retire</h4><p>Switch the old cluster off.</p></div>
</div>
<div class="mnotes">
<p>Each step is reversible until the drain.</p>
<p class="warn">The cut-over needs one 15-minute change window.</p>
</div>

---

<!-- eyebrow: Trial and PoC -->

## Prove it on ==your traffic shape.==

<div class="poc">
<div class="pc2"><span class="tag">Week 1</span><h4>Trial</h4><p>A three-node cluster in your account.</p><dl><div><dt>Scope</dt><dd>One region</dd></div><div><dt>Success</dt><dd>Zone kill, no outage</dd></div></dl><div class="term2"><span class="c"># kill a zone, watch the leaders move</span>
<span class="k">$</span> acemq chaos zone-loss --zone b
<span class="ok">leaders moved: 9 in 5.8s</span></div></div>
<div class="pc2 hi"><span class="tag">Weeks 2–6</span><h4>Joint PoC</h4><p>Your producers, your message sizes, your peak.</p><dl><div><dt>Scope</dt><dd>Card authorisations</dd></div><div><dt>Success</dt><dd>42k/s at p99 &lt; 20ms</dd></div></dl></div>
</div>

---

<!-- eyebrow: Next steps -->

## Four steps to ==a quiet Friday.==

<div class="steps">
<div class="st"><span class="num">1</span><h4>Agree scope</h4><p>Queues and success criteria.</p></div>
<div class="st"><span class="num">2</span><h4>Trial</h4><p>Three nodes, one week.</p></div>
<div class="st"><span class="num">3</span><h4>Joint PoC</h4><p>Five weeks on your shape.</p></div>
<div class="st"><span class="num">4</span><h4>Decide</h4><p>With your numbers in hand.</p></div>
</div>

---

<!-- class: on-ink -->
<!-- footer: off -->

# A lost zone should be ==a log line.==

<!-- logo: × Northwind Pay · alt=AceMQ -->

<p class="lead">Commercials to follow.</p>

<p class="contact"><a class="btn" href="#1">Book the trial</a><span class="tag orange">AceMQ Solutions</span></p>
