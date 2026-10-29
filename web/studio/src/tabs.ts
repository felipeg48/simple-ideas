/** under: path is root itself or lies inside the folder root. */
export const under = (path: string, root: string) => path === root || path.startsWith(root + "/");

/**
 * survivingTabs decides which open tabs outlive a tree refresh: a clean tab whose
 * file is gone closes; an unsaved one stays (its text exists nowhere else) and is
 * reported missing so the UI can offer to save it again or discard it.
 */
export function survivingTabs(tabs: string[], exists: (p: string) => boolean, unsaved: Set<string>) {
  const missing = tabs.filter((t) => !exists(t) && unsaved.has(t));
  return { tabs: tabs.filter((t) => exists(t) || unsaved.has(t)), missing };
}
