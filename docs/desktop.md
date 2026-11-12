# si Studio for macOS

`si Studio.app` is the Studio (`si --web`) in its own native window: no
browser tab, its own Dock icon, folders dropped on the icon open, and the
teleprompter, Play and full screen work as windows of the app.

## Install

Download `si-Studio-<version>-macos.zip` from the release, unzip it and move
`si Studio.app` to Applications. It runs on Apple silicon and Intel, macOS 13
or later.

The app is not signed with a Developer ID or notarised (decided: no code
signing on any platform), so macOS blocks the first launch. Either right-click
the app in Finder, choose **Open**, then **Open** again, or run:

```bash
xattr -dr com.apple.quarantine "/Applications/si Studio.app"
```

Builds are signed with a self-signed identity, "simple-ideas Self-Signed",
when the machine that builds has it: `make desktop-cert` creates it once in
your login keychain (the key stays there; nothing is committed), and `make
desktop-mac` then signs with it, falling back to ad-hoc signing when it is
missing. A stable identity lets macOS keep the permissions it granted the app
across updates; it does not make the app "from an identified developer", so a
downloaded copy still needs the first-open step above. Set
`SI_SIGN_IDENTITY` to sign with another identity, e.g. a Developer ID.

The Help menu opens the docs site (Documentation, Rules, Manual, Release
Notes), Report a Problem… and Request a Feature… (GitHub issue forms with the
version, "desktop app" and the OS filled in), the start gallery and the tour.
About si Studio shows the Studio's About: version, build, the si binary in use,
links and the open-source parts si is built with. The links live in one place,
si's `internal/brand`.

## Use

- On first launch it asks for a folder (decks, documents, a guide, a book, a
  site); after that it reopens the last folder. **File ▸ Open Folder…** (⌘O)
  opens another folder in another window, and so does dropping a folder on the
  Dock icon or `open -a "si Studio" path/to/folder`.
- Everything in the Studio works the same: Write, Rehearse, Present, Lab, ⌘K,
  Build, the terminal. ⌘Z ⌘X ⌘C ⌘V ⌘A reach the editor through the Edit menu;
  **View ▸ Reload** (⌘R) reloads the page.
- The teleprompter window, **Play** and **Open in a new tab** open as app
  windows; links to other sites open in your browser.
- **Present ▸ Full screen** puts the deck on the whole screen.
- **Help ▸ Start Gallery** opens the starter gallery and **Help ▸ Take the
  Tour** the Studio tour in the front window (the same as ⌘K ▸ Start gallery /
  Take the tour; a Studio URL ending in `#gallery` or `#tour` opens them too).
- Safari's Web Inspector can inspect the app (Safari ▸ Develop ▸ si Studio).

## Staying current

- **A newer si wins.** The app runs the si inside it, unless an si you
  installed is newer: `~/.bin/si`, `~/bin/si`, `~/go/bin/si`,
  `/opt/homebrew/bin/si`, `/usr/local/bin/si` or the one on PATH. **About si
  Studio** says which one runs ("using si 2.2.0 from ~/.bin/si, newer than the
  one in the app").
- **New releases.** Once a day at launch the app asks GitHub for the latest
  release (3 seconds at most, silent when offline; the answer is cached in
  `~/Library/Caches/si-studio/release.json`). When it is newer than the si in
  use, the app menu gets **Update Available — vX.Y.Z** and the Studio window a
  small note with a **Download** link; both open the release page in your
  browser. Installing is up to you: unzip and replace the app.

## How it works

| Piece | File | What |
|---|---|---|
| Shell, UI | `desktop/app_darwin.m` | Cocoa windows and menus, one WKWebView per window; new windows for `window.open` / `target=_blank`; alerts and confirms; external links to the browser |
| Shell, processes | `desktop/main.go` | per folder, starts the bundled `si --web <folder> --no-open --addr 127.0.0.1:0` (in its own process group, with the login shell's PATH: an app started from Finder gets a bare PATH), reads the Studio URL it prints, stops it when the window closes or the app quits |
| Bundle | `desktop/build-mac.sh`, `Info.plist`, `AppIcon.icns` | builds `si` and the shell for arm64 and amd64, `lipo`s them, writes the bundle, ad-hoc signs it (Apple silicon runs no unsigned code), zips it |
| Icon | `desktop/icon/gen.go` | draws the Stack mark on a dark squircle; `make desktop-icon` writes `AppIcon.icns` |

The desktop shell is its own Go module (`desktop/go.mod`): the si module keeps
no cgo and no new dependency, and `go test ./...` at the root does not build it.
It uses only Apple's frameworks (Cocoa, WebKit) through cgo; no Wails, no
Electron. Each window talks to its own `si --web` exactly like a browser tab,
with the same loopback-only, token-to-cookie security (see `docs/studio.md`).

```bash
make desktop-mac                    # desktop/dist/si Studio.app + the zip
cd desktop && go test ./...         # starts and stops a real si --web, no window
```

The tag-driven release workflow builds the app on a macOS runner and attaches
the zip to the release.

## Not yet

- Windows and Linux desktop builds (`si --web` works there today).
- Choosing the display for Present (use the green full-screen button or drag the window).

## The menu-bar icon

While si Studio is open, a stack icon sits in the menu bar. Its menu lists
every `si play` and `si serve` running on this Mac, started from the app or a
terminal, as `talk.md   127.0.0.1:7070`. Each one has:

- **Open in Browser**: the deck (or the index for a folder)
- **Teleprompter**: the deck's teleprompter in the browser (`?si-view=tp`)
- **Phone QR Codes…**: with `--remote`, the phone links as codes (`/_si/qr`,
  served on loopback only, since the links carry the key)
- **Copy Link**, **Show in Finder**
- **Stop**: stops that `si play` the way Ctrl+C does

Then **Open Folder in si Studio…** and **Quit si Studio**. Closing the last
window no longer quits the app; Quit (here or ⌘Q) does. The app does not
start at login.

How it knows: each `si play` / `si serve` writes
`~/.si/sessions/<pid>.json` (pid, folder or deck, loopback URL,
the QR page) and deletes it when it stops (`internal/serve/session.go`); the
app reads them when the menu opens, skips non-loopback ones, and deletes the
files of sessions whose process is gone (`desktop/sessions.go`). Stop only
ever signals a listed session.

