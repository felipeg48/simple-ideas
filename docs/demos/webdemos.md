---
type: demo
title: A demo that tests and films itself
app: .
viewport: 1440x900
timeout: 30s
voice: kokoro:af_heart
pronounce: si = ess eye; Northwind = north wind
transition: fade
setup: rm -rf out broken.md app
---

# A demo that tests and films itself

One Markdown script drives a web app, checks every step, and becomes a
narrated video. This one runs si's own shop example: first as a test, then
with a mistake in it, then on film.

## One script, every step checked

What it does: a demo script is a list of steps a person would take, and each one is checked as it runs.

Say: A demo script reads like a list of what a person does: go to a page, type, click, and what should appear.
- terminal "si" · window=mac theme=night backdrop=graphite
- run "sed -n '/^## Sign in/,/^## Pay/p' ../../examples/webdemo/shop.md | grep -v '^## Pay'" · wait
- expect output "click \"Sign in\""
- note "Steps a person would take" at line "type \"demo\" into \"Username\""
- wait 2s

Say: Run it as a test. Chrome performs every step, out of sight, and checks each one.
- run "clear; SHOP_PASSWORD=shop-demo si demo test ../../examples/webdemo/shop.md" · wait · within 120s
- expect output "3 passed, 0 failed"
- note "Every step, checked" at line "3 passed"
- wait 2s

## A broken step, caught

What it does: when the app does not do what the script expects, the step fails with what it expected and what it found.

Say: Now the script expects a greeting the shop never shows.
- start "sed 's/Welcome back, demo/Welcome home, demo/' ../../examples/webdemo/shop.md > broken.md && cp -R ../../examples/webdemo/app ." as copy
- stop copy · wait
- terminal "si" · window=mac theme=night backdrop=graphite
- run "SHOP_PASSWORD=shop-demo si demo test broken.md --feature 'Sign in'" · exit 1 · within 120s
- expect output "expected"

Say: The test says which step failed, what it expected, and what the page showed instead.
- note "What it expected" at line "expected:"
- wait 2500ms
- note "What it found" at line "found:"
- wait 2500ms

## The same script, on film

What it does: the steps that tested the app also film it, with a voice reading the narration.

Say: The same script, filmed. A drawn cursor follows each step, and a voice reads the narration.
- terminal "si" · window=mac theme=night backdrop=graphite
- run "SHOP_PASSWORD=shop-demo si demo run ../../examples/webdemo/shop.md --feature 'Sign in' -o out/shop" · wait · within 300s
- expect output "shop.mp4"
- note "One video, voice included" at line "shop.mp4"
- wait 2s

Say: Here is the video it made.
- browser · slide
- go /out/shop/shop.mp4
- wait 4s
