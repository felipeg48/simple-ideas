# Screen demo recorder: plan for the next big milestone

Status: **idea, being refined by the founder. Do not build yet.** Saved
2026-11-06 so the next session starts from the same understanding.

## The feature

A scripted, screen-level demo maker for **native Mac apps**:

- Give it the **app** (opened automatically) and a **script per feature**: what
  to do and what to say.
- It performs the steps on the real app windows (menus, buttons, typing),
  records the **screen or a region**, speaks the **narration** in a voice you
  pick, and produces a small **MP4** per clip, ready for YouTube or any other
  platform.
- Like Playwright, but at **screen level**, not in a browser.
- The same script also runs as a **test**: a step that fails gives feedback
  to fix the bug.

Example: an app with a menu bar; the script walks through each menu feature,
says what it does, and the video shows it happening.

## Founder decisions so far

- **Native Mac apps** first.
- A **companion app** that only talks to `si` (working name `si-recorder`;
  a better name is open: candidates *si Director*, *si Take*, *si Reel*,
  *si Stage*).

## Why it fits si

- `type: demo` scripts already hold *what to say* (`> Say:`) and *what to
  run*. Add UI steps (`open menu File › Export`, `click button Save`, `type …`)
  and that is the script format.
- The script text gives **subtitles (.srt)** and **YouTube chapters** for free.
- **Narration drives timing**: synthesize each step's audio first, measure it,
  then pace the actions to it, so every take is identical.
- **Testing**: a failed step yields a screenshot, the clip of the moment it
  broke, and the exact step: a better bug report than most people write.

## Go or a Swift companion?

Go *can* reach the Mac through cgo + Objective-C (the desktop shell in
`desktop/` already does): ScreenCaptureKit / AVFoundation for recording,
`AXUIElement` (a plain C API) for finding menus and buttons, `CGEvent` for
mouse and keyboard, `AVSpeechSynthesizer` for voices.

A **Swift companion app** is still the better fit:

- **Permissions** (Screen Recording, Accessibility) belong to an app bundle's
  identity. A CLI inherits them from whatever terminal launched it, which is
  confusing; a companion `.app` with a stable bundle id asks once, by name.
- ScreenCaptureKit, AVFoundation and the speech APIs are Swift-first (async,
  delegates); doing them through cgo is possible but painful.
- `si` stays Go and cross-platform; it talks to the companion over a local
  Unix socket with JSON lines (the same pattern as the terminal teleprompter's
  `internal/tui/sync.go`).

Caveat: macOS ties these permissions to the code signature. An ad-hoc signed
app (our current "no code signing" decision) may lose them on every rebuild and
ask again; a Developer ID signature avoids that. Revisit the signing decision
for this companion.

## Hard parts (honest)

1. **Finding things on screen.** The Accessibility API is reliable for menu
   bars, standard buttons, text fields and windows (the menu-bar case is the
   easy one); weak for custom-drawn views, canvases, games and some Electron
   apps. The alternative is AI vision ("computer use": a screenshot, then a
   model decides where to click): handles anything, but slower, paid, and not
   fully deterministic, so poor for repeatable takes and fine for exploratory
   testing. Recommendation: deterministic steps first; AI fallback only where
   asked.
2. **Permissions and a clean screen.** Screen Recording + Accessibility
   permissions; Do Not Disturb during takes; multiple monitors and Retina
   scaling; maybe record inside a VM.
3. **External tools.** Recording: ScreenCaptureKit (in the companion) or
   ffmpeg. Voices: macOS built-in (free, offline, decent Premium voices) or a
   cloud service (ElevenLabs, OpenAI TTS) with an API key. Muxing to MP4
   (H.264 + AAC): AVFoundation in the companion, or ffmpeg. Documented
   exceptions to the stdlib-only rule, like Chrome for PDFs.
4. **Platforms.** macOS first. Windows (UI Automation) and Linux (AT-SPI,
   X11 / Wayland) would each be a separate project.
5. **Safety.** It moves the mouse and types: a visible "recording" indicator
   and an obvious abort (Esc, or the mouse to a screen corner).
6. **Developing it.** Building and testing means clicking and recording on a
   real desktop, which breaks the "no windows on the founder's screen" rule.
   Needs a sandbox Mac: the `terminal-mac` VM plugin (installed 2026-11-06,
   failing to connect: `Executable not found in $PATH: stdio`) must work first.

## Phases (proposed)

| Phase | What |
|---|---|
| P1 | Record the screen or a region; narration from the script; the founder clicks while a prompter shows the next step; MP4 out |
| P2 | Automated steps: launch the app, menus, keyboard shortcuts, typing, clicks on named controls, waits |
| P3 | Checks (`window X appears`, `text Y visible`) and a test report with failure clips and screenshots |
| P4 | Optional AI-driven steps written in plain language |
| P5 | Polish: zoom on clicks, highlighted cursor, captions, chapters |

## Version

By semantic versioning this is still a minor release (nothing existing
breaks); calling it **2.0.0** is a product decision the founder may take for a
feature this size.

## Open questions for the founder

1. Which apps first: your own native Mac apps only, or also Electron apps and
   web apps in a browser? (Accessibility API vs. AI vision.) *Answered:
   native Mac apps first.*
2. Steps in plain language (AI decides what to click), a precise step syntax
   (`menu File › Export`, `click button Save`, `type …`), or precise with
   plain language as a fallback?
3. Voice: macOS built-in voices, cloud voices with an API key, or your own
   cloned voice?
4. Output: one MP4 per feature, or one video with chapters? Resolution?
   Burned-in captions?
5. Is ffmpeg (via `brew`) acceptable as a requirement, or should the companion
   do everything with Apple's frameworks?
6. "Feedback to fix bugs": a report with clips and screenshots, or GitHub
   issues filed automatically?
7. Record in a VM (clean, isolated) or on your real desktop?
8. The companion's name.
