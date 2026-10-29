import { render } from "preact";
import { App } from "./app";
import { applyTheme, storedTheme } from "./theme";

// ?theme=light|dark|system forces a palette (QA screenshots); otherwise the stored choice.
const forced = new URLSearchParams(location.search).get("theme");
applyTheme(forced === "light" || forced === "dark" || forced === "system" ? forced : storedTheme());
render(<App />, document.getElementById("app")!);
