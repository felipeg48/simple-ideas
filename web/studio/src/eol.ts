// CodeMirror always joins lines with "\n". A CRLF file is edited as LF and
// written back with its own line ending, so it never opens dirty and a save
// keeps the file's line endings.

/** eolOf is the line ending a file uses: CRLF if it has any, else LF. */
export const eolOf = (s: string) => (s.includes("\r\n") ? "\r\n" : "\n");
export const toLF = (s: string) => s.replace(/\r\n/g, "\n");
export const withEol = (s: string, eol: string) => (eol === "\n" ? s : s.replace(/\n/g, eol));
