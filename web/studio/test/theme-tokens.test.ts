import { describe, expect, it } from "vitest";
import { hex6, isThemeFile, lengthParts, sliderRange, short, themeSlug } from "../src/theme-tokens";

describe("theme view helpers", () => {
  it("knows a theme file", () => {
    expect(isThemeFile("/* si theme: Acme · extends studio */\n:root {\n}\n")).toBe(true);
    expect(isThemeFile(":root {\n  --si-accent: #e4572e;\n}\n")).toBe(true);
    expect(isThemeFile(".x { color: red }")).toBe(false);
    expect(isThemeFile("body { --si-accent: red }")).toBe(false);
  });
  it("feeds colour pickers", () => {
    expect(hex6("#E4572E")).toBe("#e4572e");
    expect(hex6("#abc")).toBe("#aabbcc");
    expect(hex6("#11223380")).toBe("#112233");
    expect(hex6("rgb(0 0 0)")).toBe("");
    expect(hex6(undefined)).toBe("");
  });
  it("splits lengths for sliders", () => {
    expect(lengthParts("3.4em")).toEqual({ n: 3.4, unit: "em" });
    expect(lengthParts("-0.02em")).toEqual({ n: -0.02, unit: "em" });
    expect(lengthParts(".55em")).toEqual({ n: 0.55, unit: "em" });
    expect(lengthParts("max(1rem, 7cqi)")).toBeNull();
    expect(lengthParts("48px")).toBeNull();
    const r = sliderRange("em", 3.1);
    expect(r.min).toBe(0);
    expect(r.max).toBeGreaterThanOrEqual(6.2);
    expect(sliderRange("em", -0.02).min).toBeLessThan(0);
  });
  it("names", () => {
    expect(short("--si-accent-2")).toBe("accent-2");
    expect(themeSlug(" Acme Brand! ")).toBe("acme-brand");
  });
});
