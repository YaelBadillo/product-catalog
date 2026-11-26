import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "vitest-axe";
import { App } from "./App";

// App brings its own providers and router, so it is rendered without
// renderWithProviders.
describe("App", () => {
  it("renders the status page with the light theme", async () => {
    const { container } = render(<App />);

    expect(
      await screen.findByRole("region", { name: "API" }),
    ).toHaveTextContent("Operational");
    expect(
      screen.getByRole("heading", { level: 1, name: "System status" }),
    ).toBeInTheDocument();
    expect(document.documentElement).toHaveAttribute(
      "data-mantine-color-scheme",
      "light",
    );
    expect(await axe(container)).toHaveNoViolations();
  });
});
