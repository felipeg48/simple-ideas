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

You can also sign it with your own identity: `codesign --force --deep -s "Developer ID Application: …" "si Studio.app"`.

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
- Safari's Web Inspector can inspect the app (Safari ▸ Develop ▸ si Studio).

## How it works

| Piece | File | What |
|---|---|---|
| Shell, UI | `desktop/app_darwin.m` | Cocoa windows and menus, one WKWebView per window; new windows for `window.open` / `target=_blank`; alerts and confirms; external links to the browser |
| Shell, processes | `desktop/main.go` | per folder, starts the bundled `si --web <folder> --no-open --addr 127.0.0.1:0` (in its own process group, with the login shell's PATH: an app started from Finder gets a bare PATH), reads the Studio URL it prints, stops it when the window closes or the app quits |
| Bundle | `desktop/build-mac.sh`, `Info.plist`, `AppIcon.icns` | builds `si` and the shell for arm64 and amd64, `lipo`s them, writes the bundle, ad-hoc signs it (Apple silicon runs no unsigned code), zips it |
| Icon | `desktop/icon/gen.go` | draws the icon (`go run desktop/icon/gen.go icon.png`, then `iconutil`) |

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
