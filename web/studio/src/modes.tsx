// The Rehearse, Present and Lab modes: the live deck with the real teleprompter
// (the deck page opened with ?si-view=tp, synced over BroadcastChannel), full
// screen, and the si play terminal drawer with its runbook.
import { useRef, useState } from "preact/hooks";
import { type DeckInfo, isSlides, pageURL } from "./api";
import { Icon } from "./navigator";

export type Mode = "write" | "rehearse" | "present" | "lab";

export function ModeView(p: { mode: Mode; deck: DeckInfo | null; decks: DeckInfo[]; slide: number | null; onPick(path: string): void }) {
  const deckFrame = useRef<HTMLIFrameElement>(null);
  const [termOpen, setTermOpen] = useState(false);
  const slides = p.decks.filter((d) => isSlides(d.kind));
  if (!p.deck || !isSlides(p.deck.kind)) {
    return (
      <div class="mode-empty">
        <div class="welcome-card">
          <h1>{p.mode === "rehearse" ? "Rehearse" : p.mode === "present" ? "Present" : "Lab"}</h1>
          <p class="mute">{slides.length ? "This mode works on slides. Pick a deck:" : "There are no slide decks in this folder yet."}</p>
          <ul class="deck-list">
            {slides.map((d) => (
              <li><button onClick={() => p.onPick(d.path)}>
                <Icon name="deck" class="k-deck" /><span class="t">{d.title}</span><span class="p">{d.path}</span>
              </button></li>
            ))}
          </ul>
        </div>
      </div>
    );
  }
  const url = pageURL(p.deck), start = p.slide ? "#" + p.slide : "";
  const tp = url + "?si-view=tp";
  const full = () => deckFrame.current?.requestFullscreen?.();
  const tpWindow = () => window.open(tp, "si-tp-" + p.deck!.path, "width=1200,height=760");
  // Lab opens the terminal drawer inside the deck as soon as it loads
  const onLoad = () => {
    if (p.mode !== "lab") return;
    try { (deckFrame.current?.contentWindow as any)?.SI?.run("terminal"); setTermOpen(true); } catch { /* not loaded */ }
  };
  const toggleTerm = () => {
    try { (deckFrame.current?.contentWindow as any)?.SI?.run("terminal"); setTermOpen((o) => !o); } catch { /* ignore */ }
  };
  return (
    <div class={"mode mode-" + p.mode}>
      <div class="mode-tool">
        <span class="mode-title"><Icon name="deck" class="k-deck" />{p.deck.title}</span>
        <span class="spacer" />
        {p.mode === "lab" && <button class="btn" onClick={toggleTerm}><Icon name="terminal" />{termOpen ? "Hide terminal" : "Show terminal"}</button>}
        {p.mode !== "present" && <button class="btn" onClick={tpWindow}><Icon name="external" />Teleprompter window</button>}
        {p.mode === "present" && <>
          <button class="btn" onClick={tpWindow}><Icon name="mic" />Open the teleprompter</button>
          <button class="btn btn-pri" onClick={full}><Icon name="play" />Full screen</button>
        </>}
      </div>
      <div class="mode-panes">
        <iframe key={p.mode + url} ref={deckFrame} class="mode-deck" src={url + start} title={`${p.deck.title}, live`}
          allow="fullscreen; clipboard-write" onLoad={onLoad} />
        {p.mode !== "present" && <iframe key={"tp" + url} class="mode-tp" src={tp} title={`Teleprompter for ${p.deck.title}`} />}
      </div>
    </div>
  );
}
