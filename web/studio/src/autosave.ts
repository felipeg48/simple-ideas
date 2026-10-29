/** createAutosave debounces save: touch() (re)arms a timer, flush() saves now if anything is pending. */
export function createAutosave(save: () => Promise<void>, delayMs = 700) {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const run = () => {
    timer = undefined;
    return save();
  };
  return {
    touch() {
      clearTimeout(timer);
      timer = setTimeout(() => void run().catch(() => {}), delayMs);
    },
    flush(): Promise<void> {
      if (timer === undefined) return Promise.resolve();
      clearTimeout(timer);
      return run();
    },
  };
}
