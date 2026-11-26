import { render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "../test/axe";
import { renderRoute } from "../test/render";
import { App } from "./App";

function header() {
  return screen.getByRole("banner");
}

describe("App", () => {
  it("renders with its own providers and the light theme", async () => {
    window.history.pushState({}, "", "/status");
    const { container } = render(<App />);

    expect(
      await screen.findByRole("region", { name: "API" }),
    ).toHaveTextContent("En funcionamiento");
    expect(document.documentElement).toHaveAttribute(
      "data-mantine-color-scheme",
      "light",
    );
    expect(await axe(container)).toHaveNoViolations();
  });
});

describe("routes", () => {
  it("redirects / to /products", async () => {
    const { router } = renderRoute("/");

    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/products");
    });
  });

  it("shows the status page in Spanish at /status", async () => {
    const { container } = renderRoute("/status");

    expect(
      await screen.findByRole("heading", {
        level: 1,
        name: "Estado del servicio",
      }),
    ).toBeInTheDocument();
    expect(
      await screen.findByRole("region", { name: "Base de datos" }),
    ).toHaveTextContent("En funcionamiento");
    expect(document.title).toBe("Estado del servicio · Catálogo de productos");
    expect(await axe(container)).toHaveNoViolations();
  });

  it("shows not found for an unknown path", async () => {
    const { user, router, container } = renderRoute("/no-existe");

    expect(
      screen.getByRole("heading", { level: 1, name: "Página no encontrada" }),
    ).toBeInTheDocument();
    expect(document.title).toBe("Página no encontrada · Catálogo de productos");
    expect(await axe(container)).toHaveNoViolations();

    await user.click(
      within(screen.getByRole("main")).getByRole("link", {
        name: "Volver al catálogo",
      }),
    );

    expect(router.state.location.pathname).toBe("/products");
  });

  it("shows the navigation header with the active link marked", () => {
    renderRoute("/status");

    const nav = within(header()).getByRole("navigation");
    expect(within(nav).getByRole("link", { name: "Catálogo" })).toHaveAttribute(
      "href",
      "/products",
    );
    expect(
      within(nav).getByRole("link", { name: "Nuevo producto" }),
    ).toHaveAttribute("href", "/products/new");
    const status = within(nav).getByRole("link", { name: "Estado" });
    expect(status).toHaveAttribute("aria-current", "page");
    expect(
      within(nav).getByRole("link", { name: "Catálogo" }),
    ).not.toHaveAttribute("aria-current");
  });
});
