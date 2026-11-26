import { DEFAULT_THEME, mergeMantineTheme } from "@mantine/core";
import { describe, expect, it } from "vitest";
import { cssVariablesResolver, theme } from "./theme";

const resolved = mergeMantineTheme(DEFAULT_THEME, theme);
const { black, white } = resolved;

function color(name: string, shade: number): string {
  return resolved.colors[name][shade];
}

/** WCAG 2.x relative luminance of a `#rrggbb` color. */
function luminance(hex: string): number {
  const [r, g, b] = [1, 3, 5].map((start) => {
    const channel = parseInt(hex.slice(start, start + 2), 16) / 255;
    return channel <= 0.04045
      ? channel / 12.92
      : ((channel + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

/** WCAG 2.x contrast ratio between two `#rrggbb` colors. */
function contrastRatio(a: string, b: string): number {
  const [lighter, darker] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (lighter + 0.05) / (darker + 0.05);
}

describe("theme", () => {
  it("uses the brand color generated from #4c5fd5 as primary, shade 6", () => {
    expect(theme.primaryColor).toBe("brand");
    expect(theme.primaryShade).toBe(6);
    expect(color("brand", 5)).toBe("#4c5fd5");
  });

  it("dims text with gray.7 and draws default borders with gray.3", () => {
    expect(cssVariablesResolver(resolved).light).toMatchObject({
      "--mantine-color-dimmed": "var(--mantine-color-gray-7)",
      "--mantine-color-default-border": "var(--mantine-color-gray-3)",
    });
  });
});

describe("palette contrast (WCAG AA)", () => {
  // [role, foreground, background], from the color table in web/CLAUDE.md.
  const textPairs: [string, string, string][] = [
    ["primary button", white, color("brand", 6)],
    ["primary button hover", white, color("brand", 7)],
    ["header", white, black],
    ["active header link", color("brand", 2), black],
    ["body", black, white],
    ["secondary text", color("gray", 7), white],
    ["secondary text on a subtle surface", color("gray", 7), color("gray", 0)],
    ["body on a subtle surface", black, color("gray", 0)],
    ["danger button", white, color("red", 9)],
    ["error alert", color("red", 9), white],
  ];

  it.each(textPairs)(
    "%s text reaches 4.5:1",
    (_role, foreground, background) => {
      expect(contrastRatio(foreground, background)).toBeGreaterThanOrEqual(4.5);
    },
  );

  it("operational status icon reaches 3:1 on white", () => {
    expect(contrastRatio(color("green", 8), white)).toBeGreaterThanOrEqual(3);
  });
});
