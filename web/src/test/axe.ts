import { configureAxe } from "vitest-axe";

/** axe limited to the WCAG 2.2 AA rules (R24). Every test imports it from here. */
export const axe = configureAxe({
  runOnly: {
    type: "tag",
    values: ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"],
  },
});
