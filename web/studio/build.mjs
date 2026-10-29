// Bundles the Studio UI into internal/assets/web/studio (embedded by the si binary):
// index.html at the root, app.js and app.css under assets/ (served at /_si/studio/assets/).
import { build } from "esbuild";
import { mkdirSync, writeFileSync } from "node:fs";

const out = new URL("../../internal/assets/web/studio/", import.meta.url).pathname;
mkdirSync(out + "assets", { recursive: true });

await build({
  entryPoints: [{ in: "src/main.tsx", out: "app" }, { in: "src/styles.css", out: "app" }],
  bundle: true,
  minify: true,
  target: "es2020",
  format: "esm",
  jsx: "automatic",
  jsxImportSource: "preact",
  outdir: out + "assets",
  legalComments: "none",
  logLevel: "info",
});

writeFileSync(out + "index.html", `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>si Studio</title>
<link rel="icon" href="data:,">
<script>try{var m=new URLSearchParams(location.search).get("theme")||localStorage.getItem("si-studio-theme")||"system";document.documentElement.dataset.theme=m==="system"?(matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light"):m}catch(e){}</script>
<link rel="stylesheet" href="/_si/studio/assets/app.css">
<script type="module" src="/_si/studio/assets/app.js"></script>
</head>
<body>
<div id="app"></div>
</body>
</html>
`);
