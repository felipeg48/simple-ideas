// The Class mode: the live class view of a course (si serve course/ --class)
// inside the Studio. Start the class serves the course on its own port with
// the class collection on; the dashboard shows here, with the link and QR
// code learners open and the keyed link for the instructor's tablet.
import { useEffect, useState } from "preact/hooks";
import { type ClassState, type DeckInfo, api, ApiError } from "./api";
import { Icon } from "./navigator";

export function ClassView(p: { decks: DeckInfo[] }) {
  const courses = p.decks.filter((d) => (d.kind as string) === "course" && d.path.split("/").pop() === "_course.md");
  const [course, setCourse] = useState<string | null>(null);
  const [st, setSt] = useState<ClassState | null>(null);
  const [lan, setLan] = useState(true);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const cur = course ?? courses[0]?.path ?? null;
  useEffect(() => {
    setSt(null); setErr("");
    if (cur) api.class(cur).then((s) => { setSt(s); if (s.running) setLan(!!s.lan); }).catch((e) => setErr(e instanceof ApiError ? e.message : "Cannot reach si."));
  }, [cur]);
  if (!cur) {
    return (
      <div class="mode-empty">
        <div class="welcome-card">
          <h1>Class</h1>
          <p class="mute">The class view follows learners through a course: who finished which lesson, quiz scores, failing checks and who is stuck. There are no courses in this folder yet (New… → Course).</p>
        </div>
      </div>
    );
  }
  const title = courses.find((c) => c.path === cur)?.title ?? cur;
  const run = async (stop: boolean) => {
    setBusy(true); setErr("");
    try { setSt(await api.classStart(cur, lan, stop)); } catch (e) { setErr(e instanceof ApiError ? e.message : "Cannot reach si."); }
    setBusy(false);
  };
  const copy = (s: string) => { void navigator.clipboard?.writeText(s).catch(() => undefined); };
  return (
    <div class="mode mode-class">
      <div class="mode-tool">
        <span class="mode-title"><Icon name="guide" class="k-guide" />
          {courses.length > 1
            ? <select aria-label="Course" value={cur} onChange={(e) => setCourse((e.target as HTMLSelectElement).value)}>
                {courses.map((c) => <option value={c.path}>{c.title}</option>)}
              </select>
            : title}
        </span>
        <span class="spacer" />
        <label class="class-lan"><input type="checkbox" checked={lan} disabled={busy} onChange={(e) => setLan((e.target as HTMLInputElement).checked)} />Learners on my network</label>
        {st?.running
          ? <>
              {!!st.lan !== lan && <button class="btn" disabled={busy} onClick={() => run(false)}>Restart</button>}
              <button class="btn" disabled={busy} onClick={() => run(true)}><Icon name="close" />Stop the class</button>
            </>
          : <button class="btn btn-pri" disabled={busy} onClick={() => run(false)}><Icon name="play" />Start the class</button>}
      </div>
      {err && <p class="class-err" role="alert">{err}</p>}
      {st?.running && st.dashboard
        ? <div class="class-panes">
            <iframe class="class-dash" src={st.dashboard} title={`Class view of ${title}`} />
            <aside class="class-share" aria-label="Share the course">
              <h2>Learners open</h2>
              <div class="class-qr" dangerouslySetInnerHTML={{ __html: st.qr ?? "" }} />
              <p class="class-link"><a href={st.learners} target="_blank" rel="noopener">{st.learners}</a>
                <button class="icon-btn" title="Copy the link" aria-label="Copy the learners' link" onClick={() => copy(st.learners!)}><Icon name="copy" /></button></p>
              {st.teacher && <>
                <h2>Class view on a tablet</h2>
                <p class="mute">A keyed link: keep it to yourself.</p>
                <p class="class-link"><code>{st.teacher}</code>
                  <button class="icon-btn" title="Copy the link" aria-label="Copy the class view link" onClick={() => copy(st.teacher!)}><Icon name="copy" /></button></p>
              </>}
              {st.note && <p class="mute">{st.note}</p>}
              <p class="mute">Events are kept in the course's .si/class/ folder, one file a day.</p>
            </aside>
          </div>
        : <div class="mode-empty">
            <div class="welcome-card">
              <h1>{title}</h1>
              <p class="mute">Start the class to serve this course to learners and follow them here live: lessons done, quiz scores, failing checks, who is stuck. With “Learners on my network” the course (only its own folder) is reachable from other devices on this network.</p>
            </div>
          </div>}
    </div>
  );
}
