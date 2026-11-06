// Writing a slide directive back into the source: the inspector's edits.

const esc = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

/**
 * setDirective sets `<!-- name: value -->` inside the slide that spans lines
 * start..end (1-based, inclusive) of text: it replaces the directive if the
 * slide has it, removes it when value is empty, or adds it under the slide's
 * heading (or at its first line when it has none).
 */
export function setDirective(text: string, start: number, end: number, name: string, value: string): string {
  const lines = text.split("\n");
  const lo = Math.max(start - 1, 0), hi = Math.min(end - 1, lines.length - 1);
  const re = new RegExp(`^\\s*<!--\\s*${esc(name)}\\s*:.*-->\\s*$`);
  const line = `<!-- ${name}: ${value.trim()} -->`;
  for (let i = lo; i <= hi; i++) {
    if (!re.test(lines[i])) continue;
    if (value.trim()) lines[i] = line;
    else {
      // drop the line, and one blank line around it so no double gap is left
      const blankAfter = lines[i + 1] !== undefined && lines[i + 1].trim() === "";
      const blankBefore = i > 0 && lines[i - 1].trim() === "";
      lines.splice(i, blankAfter && blankBefore ? 2 : 1);
    }
    return lines.join("\n");
  }
  if (!value.trim()) return text;
  let at = lo;
  while (at <= hi && lines[at].trim() === "") at++;
  if (at <= hi && /^#{1,6}\s/.test(lines[at])) lines.splice(at + 1, 0, "", line);
  else lines.splice(at, 0, line, "");
  return lines.join("\n");
}
