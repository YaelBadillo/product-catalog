import { screen, waitFor, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { apiUrl } from "../../../test/handlers";
import { axe } from "../../../test/axe";
import { renderRoute } from "../../../test/render";
import { server } from "../../../test/server";
import { messages } from "../messages";
import { buildProduct, errorResponse, productHandlers } from "../testing";

const product = buildProduct({
  id: "0193a5b0-7c1e-7000-8000-00000000abcd",
  sku: "CAF-0123",
  name: "Café molido",
  description: null,
  price_cents: 19990,
  currency: "MXN",
  stock: 5,
  category: "food",
  is_active: true,
  created_at: "2026-11-25T20:30:00Z",
  updated_at: "2026-11-26T01:05:00Z",
});
const path = `/products/${product.id}`;

/** The value shown next to a label in the details list. */
function detail(label: string) {
  const term = screen.getByText(label, { selector: "dt" });
  const value = term.nextElementSibling;
  if (!(value instanceof HTMLElement)) {
    throw new Error(`no value for ${label}`);
  }
  return value;
}

beforeAll(() => {
  vi.stubEnv("TZ", "America/Mexico_City");
});
afterAll(() => {
  vi.unstubAllEnvs();
});

describe("ProductDetailPage", () => {
  it("shows every field of the product", async () => {
    server.use(productHandlers.getOf(product));
    const { container } = renderRoute(path);

    expect(
      await screen.findByRole("heading", { level: 1, name: "Café molido" }),
    ).toBeInTheDocument();
    expect(detail(messages.labels.sku)).toHaveTextContent("CAF-0123");
    expect(detail(messages.labels.description)).toHaveTextContent(
      messages.noDescription,
    );
    expect(detail(messages.labels.price)).toHaveTextContent("$199.90 MXN");
    expect(detail(messages.labels.stock)).toHaveTextContent("5");
    expect(detail(messages.labels.category)).toHaveTextContent("Alimentos");
    expect(detail(messages.labels.status)).toHaveTextContent("Activo");
    expect(
      detail(messages.labels.status).querySelector("svg"),
      "status shows an icon next to the text",
    ).not.toBeNull();
    expect(detail(messages.labels.createdAt)).toHaveTextContent(
      "25 nov 2026, 14:30",
    );
    expect(detail(messages.labels.updatedAt)).toHaveTextContent(
      "25 nov 2026, 19:05",
    );
    expect(container).not.toHaveTextContent(product.id);
    expect(document.title).toBe("Café molido · Catálogo de productos");
    expect(await axe(container)).toHaveNoViolations();
  });

  it("links to the edit form", async () => {
    server.use(productHandlers.getOf(product));
    const { user, router } = renderRoute(path);
    await screen.findByRole("heading", { level: 1, name: "Café molido" });

    await user.click(screen.getByRole("link", { name: messages.actions.edit }));

    expect(router.state.location.pathname).toBe(`${path}/edit`);
  });

  it("shows an inactive product and its description", async () => {
    server.use(
      productHandlers.getOf({
        ...product,
        is_active: false,
        description: "Tostado medio",
        currency: "USD",
        price_cents: 129990,
      }),
    );
    renderRoute(path);

    await screen.findByRole("heading", { level: 1, name: "Café molido" });
    expect(detail(messages.labels.status)).toHaveTextContent("Inactivo");
    expect(detail(messages.labels.description)).toHaveTextContent(
      "Tostado medio",
    );
    expect(detail(messages.labels.price)).toHaveTextContent("$1,299.90 USD");
  });

  it("shows long values in full", async () => {
    const name = "N".repeat(120);
    const description = "d".repeat(2000);
    server.use(productHandlers.getOf({ ...product, name, description }));
    renderRoute(path);

    expect(
      await screen.findByRole("heading", { level: 1, name }),
    ).toBeInTheDocument();
    expect(detail(messages.labels.description).textContent).toBe(description);
  });

  it("goes back to the previous catalog view when it came from the catalog", async () => {
    server.use(productHandlers.getOf(product));
    const { user, router } = renderRoute([
      "/products?category=food&page=2",
      { pathname: path, state: { fromCatalog: true } },
    ]);

    await screen.findByRole("heading", { level: 1, name: "Café molido" });
    await user.click(
      within(screen.getByRole("main")).getByRole("button", {
        name: messages.actions.backToCatalog,
      }),
    );

    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/products");
    });
    expect(router.state.location.search).toBe("?category=food&page=2");
    expect(router.state.historyAction).toBe("POP");
  });

  it("goes to the catalog when it did not come from it", async () => {
    server.use(productHandlers.getOf(product));
    const { user, router } = renderRoute(path);

    await screen.findByRole("heading", { level: 1, name: "Café molido" });
    await user.click(
      within(screen.getByRole("main")).getByRole("button", {
        name: messages.actions.backToCatalog,
      }),
    );

    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/products");
    });
    expect(router.state.location.search).toBe("");
  });

  it("shows not found for a missing product", async () => {
    server.use(productHandlers.get(() => errorResponse(404, "NOT_FOUND")));
    const { container } = renderRoute(path);

    expect(
      await screen.findByRole("heading", {
        level: 1,
        name: messages.notFound.title,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText(messages.notFound.text)).toBeInTheDocument();
    expect(
      within(screen.getByRole("main")).getByRole("button", {
        name: messages.actions.backToCatalog,
      }),
    ).toBeInTheDocument();
    expect(await axe(container)).toHaveNoViolations();
  });

  it("shows not found for an id that is not a UUID, without a request", async () => {
    let requests = 0;
    server.use(
      http.get(apiUrl("/api/v1/products/:id"), () => {
        requests += 1;
        return HttpResponse.json(product);
      }),
    );
    renderRoute("/products/abc");

    expect(
      screen.getByRole("heading", { level: 1, name: messages.notFound.title }),
    ).toBeInTheDocument();
    await new Promise((resolve) => setTimeout(resolve, 20));
    expect(requests).toBe(0);
  });

  it("shows a loading state", async () => {
    server.use(productHandlers.pending("get"));
    const { container } = renderRoute(path);

    expect(screen.getByRole("status")).toHaveTextContent(
      messages.states.loading,
    );
    expect(await axe(container)).toHaveNoViolations();
  });

  it("shows an error with a retry that refetches", async () => {
    let fail = true;
    server.use(
      productHandlers.get(() =>
        fail
          ? errorResponse(500, "INTERNAL_ERROR")
          : HttpResponse.json(product),
      ),
    );
    const { user, container } = renderRoute(path);

    expect(
      await screen.findByText(messages.feedback.unexpected),
    ).toBeInTheDocument();
    expect(await axe(container)).toHaveNoViolations();

    fail = false;
    await user.click(
      screen.getByRole("button", { name: messages.actions.retry }),
    );

    expect(
      await screen.findByRole("heading", { level: 1, name: "Café molido" }),
    ).toBeInTheDocument();
  });
});
