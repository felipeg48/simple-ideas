---
type: demo
title: A deck, from the command line to the room
app: .
viewport: 1440x900
timeout: 15s
voice: kokoro:af_heart
pronounce: si = ess eye
setup: rm -rf talk out bad.md
---

# A deck, from the command line to the room

si turns Markdown into a talk: one command to start, one to check, one to
build, and a page that fills any screen. This script films each step and
checks it as it goes (`si demo run decks.md`, or `si demo test decks.md` for
the checks alone).

## Ask si for help

What it does: every command and its options, one keystroke away.

Say: Everything starts from one command. Ask it for help.
- terminal "si" · window=mac theme=night backdrop=graphite
- run "si --help | head -26" · wait
- expect output "markdown in, animated fluid HTML slide decks out"

Say: Play serves a deck with live reload. Build writes one HTML file you can share.
- note "Live reload" at line "si play <dir|file.md>"
- wait 2s
- note "One file per deck" at line "si [build]"
- wait 2s

Say: And new starts a talk, an article, a guide, a book, or a mind map.
- run "clear; si --help | grep -A4 'si new <name>'" · wait
- expect output "deck|article|guide|book"
- note "One command, many kinds" at line "si new <name>"

## Start a talk

What it does: a complete example talk in one command, ready to change.

Say: Start a talk. si writes a folder with a finished example deck.
- terminal "si" · window=mac theme=night backdrop=graphite
- run "si new talk" · wait
- expect output "created talk/talk.md"
- run "ls talk" · wait
- expect output "talk.md"

Say: A slide is plain Markdown: a heading, a sentence, a logo and a badge.
- run "clear; sed -n 19,27p talk/talk.md" · wait
- expect output "# The Thursday Promise"
- note "The first slide" at line "# The Thursday Promise"
- wait 2s

## Write it with live reload

What it does: save the file and the slide changes by itself.

Say: Play serves the talk, and watches the file.
- start "si new talk" as scaffold
- stop scaffold · wait
- terminal "play" · window=mac theme=night backdrop=graphite
- run "si play talk --addr 127.0.0.1:7391"
- wait for output "serving talk"
- note "Watching for changes" at line "live reload on"

Say: Here is the first slide, in the browser.
- browser
- go http://127.0.0.1:7391/talk.html
- expect text "The Thursday Promise"
- wait 1s

Say: Change the title in the file, and the slide follows.
- terminal "edit" · window=mac theme=night backdrop=graphite
- run "sed -i.bak 's/^# The Thursday Promise/# Our Thursday Promise/' talk/talk.md" · wait
- browser
- expect text "Our Thursday Promise"
- wait 2s

## Check before you share

What it does: a mistake caught with its file and line, fixed, and checked again.

Say: Here is a slide with a typo in one of its directives.
- start "printf '%s\n' '# Launch day' '' '<!-- layuot: center -->' '' 'Ship it.' > bad.md" as draft
- stop draft · wait
- terminal "si" · window=mac theme=night backdrop=graphite
- run "cat bad.md" · wait
- expect output "layuot"
- note "A typo" at line "layuot"

Say: Check finds it, says which line, and lists the directives it knows.
- run "si check bad.md" · exit 1
- expect output "unknown directive"
- note "Line 3, and why" at line "bad.md:3"
- wait 2s

Say: Fix the word, check again, and the deck is clean.
- run "clear; sed -i.bak 's/layuot/layout/' bad.md && si check bad.md" · wait
- expect output "OK"

## Build and share

What it does: one self-contained HTML file per deck, that works offline and fills any screen.

Say: Build writes every deck in the folder to plain HTML.
- start "si new talk" as scaffold
- stop scaffold · wait
- terminal "si" · window=mac theme=night backdrop=graphite
- run "si build talk -o out" · wait
- expect output "talk.html"
- note "One file, everything inside" at line "talk.html"
- wait 2s

Say: Open the file. The title slide fills the screen.
- browser
- go /out/talk.html
- expect text "The Thursday Promise"
- wait 2s

Say: One picture tells the story. Its parts change from slide to slide.
- go /out/talk.html#5
- expect text "The journey on one screen"
- wait 2s
- press ArrowRight
- expect text "Payment: declined, then paid"
- wait 1s

Say: A payment is declined, retried, and paid, one click at a time.
- press ArrowRight
- wait 1500ms
- press ArrowRight
- wait 1500ms
- press ArrowRight
- wait 1s

## Print and hand out

What it does: a PDF of the slides, and a handout for the audience, from the same file.

Say: The same build can print the slides, and a handout for the room.
- start "si new talk" as scaffold
- stop scaffold · wait
- terminal "si" · window=mac theme=night backdrop=graphite
- run "si build talk/talk.md -o out --pdf --handout" · wait · within 180s
- run "clear; ls out out/pdfs out/handouts" · wait
- expect output "talk.pdf"
- note "Slides as PDF" at line "out/pdfs"
- wait 2s

Say: The index page lists every deck, with its PDF beside it.
- browser
- go /out/index.html
- expect text "The Thursday Promise"
- wait 3s
