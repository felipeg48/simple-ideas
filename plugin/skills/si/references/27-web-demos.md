# 27. Web demos

A demo script (§22) with an `app:` in its front matter is a **web demo**: si
opens the web app in headless Chrome and performs each feature the way a
person would, step by step, checking what the page shows. `si demo test`
runs it as a test, with a report and an exit code for CI. (Narrated videos of
the same script come in a later version.)

```markdown
---
type: demo
title: Northwind Shop, from sign-in to receipt
app: app                # a URL, or the app's folder next to the script
viewport: desktop       # 1440x900; also phone (400x800), tablet (820x1180), WxH
timeout: 5s             # how long a step waits for its element
setup: ./seed.sh        # optional: run before each feature, in the script's folder
---

## Pay for an order

What it does: a customer picks Express shipping and pays with a card.

Say: Sign in with the demo account.
- go /
- type "demo" into "Username"
- type $SHOP_PASSWORD into "Password"
- click "Sign in"
- expect text "Welcome back, demo"

Say: A test card goes through in a second.
- click "Order #4711"
- select "Express" in "Shipping"
- check "Gift wrap"
- click role=button "Pay"
- type "4242 4242 4242 4242" into "Card number"
- click "Confirm"
- wait for "Paid"
- expect url /receipt.html?order=4711
- expect no errors
```

- Each `##` heading is a **feature**: one test case, run in a fresh browser
  (no cookies or storage from the one before), after `setup:` and with the
  app's first page open.
- `What it does:` describes the feature (the report's summary). Each `Say:`
  line is the narration of the steps under it, up to the next `Say:`.
  `Notes:` and `Points:` keep their usual meaning and are not steps.
- Steps are list items. A step waits up to `timeout:` for its element or
  condition, then fails with what it expected, what it found and the closest
  elements it saw.

| Step | What it does |
|---|---|
| `go /path` (or a full URL) | open a page of the app and wait for it to load |
| `click "Pay"` | a real mouse click at the element's centre |
| `type "text" into "Field"` · `type $VAR into "Field"` | replace the field's text; `$VAR` comes from the environment and is never printed (`••••` in the report) |
| `press Enter` · `press cmd+k` | a key or a shortcut (alt, ctrl, cmd, shift) |
| `select "Express" in "Shipping"` | an option of a `<select>` (or a listbox) |
| `check "Gift wrap"` · `uncheck …` | a checkbox or switch |
| `hover "Help"` · `scroll to "Footer"` | move the mouse over it; scroll it into view |
| `wait 1s` · `wait for "Saved"` | a pause; or until something appears |
| `note "Username: demo" at "Username"` | a callout for the video (`· for 3s`, `· position right`); a test checks its element is there |
| `expect text "Paid"` | the page shows these words |
| `expect url /orders/4711/receipt` | the address (a path, or a full URL; a `?…` or `#…` after it is fine) |
| `expect "Pay" disabled` | `visible`, `hidden`, `enabled`, `disabled`, `checked`, `unchecked` |
| `expect no errors` | no console errors and no failed requests (HTTP 400 and above) so far |
| `screenshot receipt` | a named picture in the report |

**Elements** are found the way a person names them: `"Pay"` matches visible
text, a `<label>`, an `aria-label`, a placeholder, a title or an alt text
(exact first, then any case, then a part). `#id` and `[data-testid=pay]` are
CSS selectors; `role=button "Pay"` narrows by role. Visible, enabled
elements come first; text inside a button means the button.

```bash
SHOP_PASSWORD=shop-demo si demo test shop.md
si demo test shop.md --feature "Pay for an order" --junit junit.xml -o out/
```

```text
Pay for an order
  ✓ go /  0.06s
  ✓ type "demo" into "Username"  0.01s
  ✓ type $SHOP_PASSWORD into "Password"  0.02s
  ✗ expect text "10% off"  5.07s
      expected: the page shows "10% off"
      found:    the page shows: Northwind Shop ? Order #4711 … That coupon is not valid. …
si demo: 0 passed, 1 failed in 6.9s; report: demo-out/shop/report.html
```

- `si demo test` writes `report.html` (one page, screenshots inlined) and
  `report.json` into `demo-out/<script>/`, exits 1 when a feature failed,
  and with `--junit file.xml` writes JUnit XML (one test case per feature).
  `si demo run` is reserved for the videos; for now it runs the test.
- A failed step stops its feature; the report shows expected and found, the
  closest elements, a screenshot, and the page's console errors and failed
  requests.
- `si check` reports a bad step with its line, and warns about a `$VAR` that
  is not set. The script can also be AsciiDoc, or JSON for tools:
  `{"type": "demo", "app": "…", "features": [{"name": "…", "what": "…",
  "blocks": [{"say": "…", "steps": ["go /", "click \"Pay\""]}]}]}`.
- Needs Google Chrome or Chromium (`si doctor`).

Example: `examples/webdemo/` (a small shop, `shop.md`, and `broken.md`, whose
coupon does not work).
