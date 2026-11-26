import { screen, waitFor, within } from "@testing-library/react";
import { delay, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";
import { axe } from "../../../test/axe";
import { renderRoute } from "../../../test/render";
import { server } from "../../../test/server";
import { messages } from "../messages";
import {
  buildProduct,
  chooseOption,
  errorResponse,
  field,
  productHandlers,
} from "../testing";
import type { Product, UpdateProductRequest } from "../types";

const product = buildProduct({
  id: "0193a5b0-7c1e-7000-8000-0000000000ed",
  sku: "CAF-0123",
  name: "Café molido",
  description: "Tostado medio",
  price_cents: 19990,
  currency: "MXN",
  stock: 5,
  category: "food",
  is_active: true,
  version: 3,
});
const path = `/products/${product.id}/edit`;

function save() {
  return screen.getByRole("button", { name: messages.actions.save });
}

/** Serves GET with `current()` and records PATCH bodies, answered by `respond`. */
function serveEdit(
  respond: (body: UpdateProductRequest) => Response | Promise<Response>,
  current: () => Product = () => product,
) {
  const bodies: UpdateProductRequest[] = [];
  server.use(
    productHandlers.get(() => HttpResponse.json(current())),
    productHandlers.update(async ({ request }) => {
      const body = (await request.json()) as UpdateProductRequest;
      bodies.push(body);
      return respond(body);
    }),
  );
  return bodies;
}

async function loaded() {
  await screen.findByRole("heading", { level: 1, name: "Editar Café molido" });
}

describe("ProductEditPage", () => {
  it("prefills the form with a read-only SKU and nothing to save yet", async () => {
    serveEdit(() => HttpResponse.json(product));
    const { container } = renderRoute(path);
    await loaded();

    expect(field(messages.labels.sku)).toHaveValue("CAF-0123");
    expect(field(messages.labels.sku)).toHaveAttribute("readonly");
    expect(field(messages.labels.name)).toHaveValue("Café molido");
    expect(field(messages.labels.description)).toHaveValue("Tostado medio");
    expect(field(messages.labels.price)).toHaveValue("199.9");
    expect(field(messages.labels.currency)).toHaveValue("MXN");
    expect(field(messages.labels.stock)).toHaveValue("5");
    expect(field(messages.labels.category)).toHaveValue("Alimentos");
    expect(
      screen.getByRole("switch", { name: messages.labels.isActive }),
    ).toBeChecked();
    expect(save()).toBeDisabled();
    expect(document.title).toBe("Editar Café molido · Catálogo de productos");
    expect(await axe(container)).toHaveNoViolations();
  });

  it("saves only the changed fields and opens the updated detail", async () => {
    const updated = { ...product, stock: 7, description: null, version: 4 };
    const bodies = serveEdit(() => HttpResponse.json(updated));
    const { user, router } = renderRoute(path);
    await loaded();

    await user.clear(field(messages.labels.stock));
    await user.type(field(messages.labels.stock), "7");
    await user.clear(field(messages.labels.description));
    await user.click(save());

    expect(
      await screen.findByText(messages.feedback.saved),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(router.state.location.pathname).toBe(`/products/${product.id}`);
    });
    expect(bodies).toEqual([{ version: 3, stock: 7, description: "" }]);
    expect(await screen.findByText(messages.noDescription)).toBeInTheDocument();
  });

  it("keeps the amount when the currency changes", async () => {
    const bodies = serveEdit(() =>
      HttpResponse.json({ ...product, currency: "USD" }),
    );
    const { user } = renderRoute(path);
    await loaded();

    await chooseOption(user, messages.labels.currency, "USD");
    await user.click(save());

    await waitFor(() => {
      expect(bodies).toEqual([{ version: 3, currency: "USD" }]);
    });
  });

  it("disables save while the request is pending", async () => {
    serveEdit(async () => {
      await delay("infinite");
      return HttpResponse.json(product);
    });
    const { user } = renderRoute(path);
    await loaded();

    await user.type(field(messages.labels.name), " tostado");
    await user.click(save());

    await waitFor(() => {
      expect(save()).toBeDisabled();
    });
  });

  it("shows client and server errors on their fields", async () => {
    const bodies = serveEdit(() =>
      errorResponse(422, "VALIDATION_ERROR", [
        { field: "stock", message: "x" },
      ]),
    );
    const { user } = renderRoute(path);
    await loaded();

    await user.clear(field(messages.labels.name));
    await user.type(field(messages.labels.name), "a");
    await user.click(save());

    expect(field(messages.labels.name)).toHaveAccessibleDescription(
      messages.rules.name,
    );
    expect(bodies).toEqual([]);

    await user.type(field(messages.labels.name), "zúcar");
    await user.click(save());

    await waitFor(() => {
      expect(field(messages.labels.stock)).toHaveAccessibleDescription(
        messages.rules.stock,
      );
    });
  });

  it("warns about a concurrent change, keeps the values and reloads", async () => {
    let current = product;
    serveEdit(
      () => errorResponse(409, "VERSION_CONFLICT"),
      () => current,
    );
    const { user, container } = renderRoute(path);
    await loaded();

    await user.clear(field(messages.labels.stock));
    await user.type(field(messages.labels.stock), "9");
    await user.click(save());

    const alert = await screen.findByText(messages.feedback.conflict);
    expect(field(messages.labels.stock)).toHaveValue("9");
    expect(await axe(container)).toHaveNoViolations();

    current = { ...product, stock: 2, name: "Café de olla", version: 4 };
    await user.click(
      within(alert.closest("[role='alert']") ?? document.body).getByRole(
        "button",
        {
          name: messages.actions.reload,
        },
      ),
    );

    await waitFor(() => {
      expect(field(messages.labels.stock)).toHaveValue("2");
    });
    expect(field(messages.labels.name)).toHaveValue("Café de olla");
    expect(
      screen.queryByText(messages.feedback.conflict),
    ).not.toBeInTheDocument();
    expect(save()).toBeDisabled();
  });

  it("shows not found when the product was deleted before saving", async () => {
    serveEdit(() => errorResponse(404, "NOT_FOUND"));
    const { user } = renderRoute(path);
    await loaded();

    await user.type(field(messages.labels.name), " tostado");
    await user.click(save());

    expect(
      await screen.findByRole("heading", {
        level: 1,
        name: messages.notFound.title,
      }),
    ).toBeInTheDocument();
  });

  it("shows loading, not found and error states on load", async () => {
    server.use(productHandlers.pending("get"));
    const loading = renderRoute(path);
    expect(screen.getByRole("status")).toHaveTextContent(
      messages.states.loading,
    );
    expect(await axe(loading.container)).toHaveNoViolations();
    loading.unmount();

    server.use(productHandlers.get(() => errorResponse(404, "NOT_FOUND")));
    const missing = renderRoute(path);
    expect(
      await screen.findByRole("heading", {
        level: 1,
        name: messages.notFound.title,
      }),
    ).toBeInTheDocument();
    missing.unmount();

    server.use(productHandlers.get(() => errorResponse(500, "INTERNAL_ERROR")));
    const failed = renderRoute(path);
    expect(
      await screen.findByRole("heading", {
        name: messages.feedback.unexpected,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: messages.actions.retry }),
    ).toBeInTheDocument();
    expect(await axe(failed.container)).toHaveNoViolations();
  });
});
