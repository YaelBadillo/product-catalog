import { screen, waitFor, within } from "@testing-library/react";
import { delay, HttpResponse } from "msw";
import { afterEach, describe, expect, it, vi } from "vitest";
import { axe } from "../../../test/axe";
import { renderRoute } from "../../../test/render";
import { server } from "../../../test/server";
import { messages } from "../messages";
import {
  buildList,
  buildProduct,
  chooseOption,
  errorResponse,
  field,
  productHandlers,
} from "../testing";
import type { ProductList } from "../types";

const cafe = buildProduct({
  id: "0193a5b0-7c1e-7000-8000-000000000001",
  sku: "CAF-0123",
  name: "Café molido",
  price_cents: 129990,
  currency: "MXN",
  stock: 5,
  category: "food",
  is_active: true,
});
const silla = buildProduct({
  id: "0193a5b0-7c1e-7000-8000-000000000002",
  sku: "SIL-0001",
  name: "Silla plegable",
  price_cents: 4500,
  currency: "USD",
  stock: 0,
  category: "home",
  is_active: false,
});

/** Records the query of every list request and answers with `respond`. */
function serveList(
  respond: (query: URLSearchParams) => ProductList | Response,
) {
  const queries: URLSearchParams[] = [];
  server.use(
    productHandlers.list(({ request }) => {
      const query = new URL(request.url).searchParams;
      queries.push(query);
      const result = respond(query);
      return result instanceof Response ? result : HttpResponse.json(result);
    }),
  );
  return {
    queries,
    last: () => queries.at(-1) ?? new URLSearchParams(),
  };
}

/** Makes `(min-width: 48em)` match, as on desktop. */
function mockDesktopLayout() {
  // Only what Mantine 9 uses; the deprecated listener methods are left out.
  vi.spyOn(window, "matchMedia").mockImplementation(
    (query: string) =>
      ({
        matches: query.includes("min-width"),
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }) as unknown as MediaQueryList,
  );
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("ProductListPage", () => {
  describe("rows", () => {
    it("shows products as a table on wide screens", async () => {
      mockDesktopLayout();
      serveList(() => buildList([cafe, silla]));
      const { container } = renderRoute("/products");

      const table = await screen.findByRole("table");
      const headers = within(table)
        .getAllByRole("columnheader")
        .map((h) => h.textContent);
      expect(headers).toEqual([
        "SKU",
        "Nombre",
        "Categoría",
        "Precio",
        "Inventario",
        "Estado",
        "Acciones",
      ]);
      const row = within(table).getByRole("row", { name: /Café molido/ });
      expect(row).toHaveTextContent("CAF-0123");
      expect(row).toHaveTextContent("Alimentos");
      expect(row).toHaveTextContent("$1,299.90 MXN");
      expect(row).toHaveTextContent("Activo");
      expect(
        within(table).getByRole("row", { name: /Silla plegable/ }),
      ).toHaveTextContent("$45.00 USD");
      expect(
        within(table).getByRole("row", { name: /Silla plegable/ }),
      ).toHaveTextContent("Inactivo");
      expect(
        screen.getByRole("heading", { level: 1, name: "Catálogo" }),
      ).toBeInTheDocument();
      expect(document.title).toBe("Catálogo · Catálogo de productos");
      expect(await axe(container)).toHaveNoViolations();
    });

    it("shows products as cards on narrow screens", async () => {
      serveList(() => buildList([cafe, silla]));
      const { container } = renderRoute("/products");

      const card = await screen.findByRole("article", { name: "Café molido" });
      expect(screen.queryByRole("table")).not.toBeInTheDocument();
      expect(card).toHaveTextContent("CAF-0123");
      expect(card).toHaveTextContent("$1,299.90 MXN");
      expect(card).toHaveTextContent("Alimentos");
      expect(card).toHaveTextContent("Activo");
      expect(await axe(container)).toHaveNoViolations();
    });

    it("keeps a long name complete in its accessible name", async () => {
      mockDesktopLayout();
      const name = "N".repeat(120);
      serveList(() => buildList([{ ...cafe, name }]));
      renderRoute("/products");

      expect(await screen.findByRole("link", { name })).toBeInTheDocument();
    });

    it("announces how many products match", async () => {
      serveList(() => buildList([cafe, silla], { total: 41 }));
      renderRoute("/products");

      await screen.findByRole("article", { name: "Café molido" });
      expect(screen.getByText("41 productos")).toHaveAttribute(
        "aria-live",
        "polite",
      );
    });
  });

  describe("URL and filters", () => {
    it("sends the URL's view as the request", async () => {
      const list = serveList(() => buildList([cafe]));
      renderRoute(
        "/products?category=food&is_active=true&currency=MXN&min_price=100&max_price=499.90&q=caf%C3%A9&sort=price&order=asc&page=2",
      );

      await screen.findByRole("article", { name: "Café molido" });
      expect(Object.fromEntries(list.last())).toEqual({
        page: "2",
        page_size: "20",
        category: "food",
        is_active: "true",
        currency: "MXN",
        min_price: "10000",
        max_price: "49990",
        q: "café",
        sort: "price",
        order: "asc",
      });
      expect(field(messages.labels.search)).toHaveValue("café");
    });

    it("ignores invalid values in the link", async () => {
      const list = serveList(() => buildList([cafe]));
      renderRoute("/products?category=toys&page=0&min_price=10&sort=color");

      await screen.findByRole("article", { name: "Café molido" });
      expect(Object.fromEntries(list.last())).toEqual({
        page: "1",
        page_size: "20",
        sort: "created_at",
        order: "desc",
      });
    });

    it("updates the URL and the request for each filter, back to page 1", async () => {
      const list = serveList(() => buildList([cafe], { total: 60 }));
      const { user, router } = renderRoute("/products?page=3");
      await screen.findByRole("article", { name: "Café molido" });

      await chooseOption(user, messages.labels.category, "Alimentos");
      await waitFor(() => {
        expect(router.state.location.search).toBe("?category=food");
      });

      await chooseOption(user, messages.labels.status, "Inactivo");
      await chooseOption(user, messages.labels.currency, "USD");
      await chooseOption(user, messages.labels.sort, "Precio");
      await chooseOption(user, messages.labels.order, "Ascendente");

      await waitFor(() => {
        expect(list.last().get("order")).toBe("asc");
      });
      expect(Object.fromEntries(list.last())).toMatchObject({
        page: "1",
        category: "food",
        is_active: "false",
        currency: "USD",
        sort: "price",
        order: "asc",
      });
      expect(router.state.historyAction).toBe("PUSH");
    });

    it("searches after a pause, and Back returns to the view before it", async () => {
      const list = serveList((query) =>
        buildList(query.get("q") === "cafe" ? [cafe] : [cafe, silla]),
      );
      const { user, router } = renderRoute("/products?category=food");
      await screen.findByRole("article", { name: "Silla plegable" });

      await user.type(field(messages.labels.search), "cafe");

      await waitFor(() => {
        expect(router.state.location.search).toBe("?category=food&q=cafe");
      });
      await waitFor(() => {
        expect(
          screen.queryByRole("article", { name: "Silla plegable" }),
        ).not.toBeInTheDocument();
      });
      expect(list.queries.filter((q) => q.has("q"))).toHaveLength(1);

      await router.navigate(-1);

      await waitFor(() => {
        expect(router.state.location.search).toBe("?category=food");
      });
      await waitFor(() => {
        expect(field(messages.labels.search)).toHaveValue("");
      });
    });

    it("applies the search at once on Enter", async () => {
      const list = serveList(() => buildList([cafe]));
      const { user } = renderRoute("/products");
      await screen.findByRole("article", { name: "Café molido" });

      await user.type(field(messages.labels.search), "silla{Enter}");

      expect(list.last().get("q")).toBe("silla");
    });

    it("disables the price range until a currency is chosen", async () => {
      serveList(() => buildList([cafe]));
      const { user, router } = renderRoute("/products");
      await screen.findByRole("article", { name: "Café molido" });

      expect(field(messages.labels.minPrice)).toBeDisabled();
      expect(field(messages.labels.maxPrice)).toBeDisabled();
      expect(
        screen.getAllByText(messages.hints.priceRangeNeedsCurrency).length,
      ).toBeGreaterThan(0);

      await chooseOption(user, messages.labels.currency, "MXN");
      await user.type(field(messages.labels.minPrice), "100");
      await user.type(field(messages.labels.maxPrice), "500");

      await waitFor(() => {
        expect(router.state.location.search).toBe(
          "?currency=MXN&min_price=100&max_price=500",
        );
      });

      await chooseOption(user, messages.labels.currency, "Todas");

      await waitFor(() => {
        expect(router.state.location.search).toBe("");
      });
      expect(field(messages.labels.minPrice)).toHaveValue("");
      expect(field(messages.labels.minPrice)).toBeDisabled();
    });

    it("rejects min > max without sending a request", async () => {
      const list = serveList(() => buildList([cafe]));
      const { user, router } = renderRoute("/products?currency=MXN");
      await screen.findByRole("article", { name: "Café molido" });

      await user.type(field(messages.labels.minPrice), "500");
      await waitFor(() => {
        expect(list.last().get("min_price")).toBe("50000");
      });
      const requests = list.queries.length;
      await user.type(field(messages.labels.maxPrice), "100");

      await waitFor(() => {
        expect(field(messages.labels.maxPrice)).toHaveAccessibleDescription(
          messages.rules.priceRange,
        );
      });
      await new Promise((resolve) => setTimeout(resolve, 400));
      expect(router.state.location.search).toBe("?currency=MXN&min_price=500");
      expect(list.queries).toHaveLength(requests);
    });

    it("moves between pages", async () => {
      const list = serveList((query) =>
        buildList([query.get("page") === "2" ? silla : cafe], {
          total: 41,
          page: Number(query.get("page")),
        }),
      );
      const { user, router } = renderRoute("/products");
      await screen.findByRole("article", { name: "Café molido" });

      await user.click(
        within(
          screen.getByRole("navigation", { name: messages.labels.pagination }),
        ).getByRole("button", { name: "2" }),
      );

      expect(
        await screen.findByRole("article", { name: "Silla plegable" }),
      ).toBeInTheDocument();
      expect(router.state.location.search).toBe("?page=2");
      expect(list.last().get("page")).toBe("2");
    });
  });

  describe("states", () => {
    it("shows a loading state", async () => {
      server.use(productHandlers.pending("get", "/api/v1/products"));
      const { container } = renderRoute("/products");

      expect(screen.getByRole("status")).toHaveTextContent(
        messages.states.loading,
      );
      expect(await axe(container)).toHaveNoViolations();
    });

    it("shows the empty catalog with a link to create a product", async () => {
      serveList(() => buildList([]));
      const { user, router, container } = renderRoute("/products");

      expect(
        await screen.findByRole("heading", {
          name: messages.states.emptyCatalog,
        }),
      ).toBeInTheDocument();
      expect(await axe(container)).toHaveNoViolations();

      await user.click(
        within(screen.getByRole("main")).getByRole("link", {
          name: messages.actions.create,
        }),
      );
      expect(router.state.location.pathname).toBe("/products/new");
    });

    it("shows no results with a way to clear the filters", async () => {
      const list = serveList((query) =>
        buildList(query.has("category") ? [] : [cafe]),
      );
      const { user, router, container } = renderRoute(
        "/products?category=home&q=zzz&sort=name",
      );

      expect(
        await screen.findByRole("heading", { name: messages.states.noResults }),
      ).toBeInTheDocument();
      expect(await axe(container)).toHaveNoViolations();

      await user.click(
        screen.getByRole("button", { name: messages.actions.clearFilters }),
      );

      await waitFor(() => {
        expect(router.state.location.search).toBe("?sort=name");
      });
      expect(
        await screen.findByRole("article", { name: "Café molido" }),
      ).toBeInTheDocument();
      expect(list.last().has("q")).toBe(false);
      expect(field(messages.labels.search)).toHaveValue("");
    });

    it("shows an out-of-range page with a way to the first page", async () => {
      serveList((query) =>
        query.get("page") === "9"
          ? buildList([], { total: 12, page: 9 })
          : buildList([cafe], { total: 12 }),
      );
      const { user, router, container } = renderRoute("/products?page=9");

      expect(
        await screen.findByRole("heading", {
          name: messages.states.outOfRange,
        }),
      ).toBeInTheDocument();
      expect(await axe(container)).toHaveNoViolations();

      await user.click(
        screen.getByRole("button", { name: messages.actions.firstPage }),
      );

      expect(
        await screen.findByRole("article", { name: "Café molido" }),
      ).toBeInTheDocument();
      expect(router.state.location.search).toBe("");
    });

    it.each([
      [500, "INTERNAL_ERROR", messages.feedback.unexpected],
      [429, "RATE_LIMITED", messages.feedback.rateLimited],
    ] as const)("shows a %i with a retry", async (status, code, text) => {
      let fail = true;
      serveList(() => (fail ? errorResponse(status, code) : buildList([cafe])));
      const { user, container } = renderRoute("/products");

      expect(
        await screen.findByRole("heading", { name: text }),
      ).toBeInTheDocument();
      expect(await axe(container)).toHaveNoViolations();

      fail = false;
      await user.click(
        screen.getByRole("button", { name: messages.actions.retry }),
      );

      expect(
        await screen.findByRole("article", { name: "Café molido" }),
      ).toBeInTheDocument();
    });
  });

  describe("navigation", () => {
    it("opens a product, and Back restores the catalog view", async () => {
      mockDesktopLayout();
      serveList(() => buildList([cafe]));
      server.use(productHandlers.getOf(cafe));
      const { user, router } = renderRoute("/products?category=food&sort=name");
      const table = await screen.findByRole("table");

      await user.click(
        within(table).getByRole("link", { name: "Café molido" }),
      );

      expect(
        await screen.findByRole("heading", { level: 1, name: "Café molido" }),
      ).toBeInTheDocument();
      await user.click(
        screen.getByRole("button", { name: messages.actions.backToCatalog }),
      );

      await waitFor(() => {
        expect(router.state.location.search).toBe("?category=food&sort=name");
      });
      expect(await screen.findByRole("table")).toBeInTheDocument();
    });

    it.each([
      ["table", true],
      ["cards", false],
    ])("offers an edit action per product in the %s", async (_, desktop) => {
      if (desktop) {
        mockDesktopLayout();
      }
      serveList(() => buildList([cafe, silla]));
      const { user, router } = renderRoute("/products");
      await screen.findAllByRole("link", { name: "Café molido" });

      await user.click(
        screen.getByRole("link", { name: "Editar Silla plegable" }),
      );

      expect(router.state.location.pathname).toBe(`/products/${silla.id}/edit`);
    });

    it("keeps the previous page on screen while the next one loads", async () => {
      serveList(() => buildList([cafe], { total: 41 }));
      const { user } = renderRoute("/products");
      await screen.findByRole("article", { name: "Café molido" });
      server.use(
        productHandlers.list(async () => {
          await delay("infinite");
          return HttpResponse.json(buildList([]));
        }),
      );

      await user.click(
        within(
          screen.getByRole("navigation", { name: messages.labels.pagination }),
        ).getByRole("button", { name: "2" }),
      );

      expect(
        screen.getByRole("article", { name: "Café molido" }),
      ).toBeInTheDocument();
    });
  });
});
