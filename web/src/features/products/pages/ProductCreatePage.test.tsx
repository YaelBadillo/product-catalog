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
import type { CreateProductRequest } from "../types";

const created = buildProduct({ sku: "ABC-123", name: "Té verde" });

function submit() {
  return screen.getByRole("button", { name: messages.actions.create });
}

/** Records every POST body and answers with `respond`. */
function capturePost(respond: () => Response | Promise<Response>) {
  const bodies: CreateProductRequest[] = [];
  server.use(
    productHandlers.create(async ({ request }) => {
      bodies.push((await request.json()) as CreateProductRequest);
      return respond();
    }),
    productHandlers.getOf(created),
  );
  return bodies;
}

async function fillValidForm(user: ReturnType<typeof renderRoute>["user"]) {
  await user.type(field(messages.labels.sku), "  abc-123 ");
  await user.type(field(messages.labels.name), "Té verde");
  await user.type(field(messages.labels.price), "1299.90");
  await user.type(field(messages.labels.stock), "3");
  await chooseOption(user, messages.labels.category, "Alimentos");
}

describe("ProductCreatePage", () => {
  it("starts with MXN, active and no category", async () => {
    const { container } = renderRoute("/products/new");

    expect(
      screen.getByRole("heading", { level: 1, name: "Nuevo producto" }),
    ).toBeInTheDocument();
    expect(field(messages.labels.currency)).toHaveValue("MXN");
    expect(
      screen.getByRole("switch", { name: messages.labels.isActive }),
    ).toBeChecked();
    expect(field(messages.labels.category)).toHaveValue("");
    expect(document.title).toBe("Nuevo producto · Catálogo de productos");
    expect(await axe(container)).toHaveNoViolations();
  });

  it("creates the product and opens its detail", async () => {
    const bodies = capturePost(() =>
      HttpResponse.json(created, {
        status: 201,
        headers: { Location: `/api/v1/products/${created.id}` },
      }),
    );
    const { user, router } = renderRoute("/products/new");

    await fillValidForm(user);
    await user.click(submit());

    expect(
      await screen.findByText(messages.feedback.created),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(router.state.location.pathname).toBe(`/products/${created.id}`);
    });
    expect(bodies).toEqual([
      {
        sku: "ABC-123",
        name: "Té verde",
        price_cents: 129990,
        currency: "MXN",
        stock: 3,
        category: "food",
        is_active: true,
      },
    ]);
    expect(
      await screen.findByRole("heading", { level: 1, name: "Té verde" }),
    ).toBeInTheDocument();
  });

  it("shows client errors on each field and focuses the first one", async () => {
    const bodies = capturePost(() =>
      HttpResponse.json(created, { status: 201 }),
    );
    const { user, container } = renderRoute("/products/new");

    await user.click(submit());

    expect(field(messages.labels.sku)).toHaveAccessibleDescription(
      messages.rules.skuRequired,
    );
    expect(field(messages.labels.name)).toHaveAccessibleDescription(
      messages.rules.nameRequired,
    );
    expect(field(messages.labels.price)).toHaveAccessibleDescription(
      messages.rules.priceRequired,
    );
    expect(field(messages.labels.stock)).toHaveAccessibleDescription(
      messages.rules.stockRequired,
    );
    expect(field(messages.labels.category)).toHaveAccessibleDescription(
      messages.rules.category,
    );
    expect(field(messages.labels.sku)).toHaveFocus();
    expect(bodies).toEqual([]);
    expect(await axe(container)).toHaveNoViolations();
  });

  it("maps a 422 to the Spanish rule of each field and keeps the values", async () => {
    capturePost(() =>
      errorResponse(422, "VALIDATION_ERROR", [
        { field: "name", message: "must be 2 to 120 characters" },
        {
          field: "price_cents",
          message: "must be between 1 and 99999999999 cents",
        },
      ]),
    );
    const { user } = renderRoute("/products/new");

    await fillValidForm(user);
    await user.click(submit());

    await waitFor(() => {
      expect(field(messages.labels.name)).toHaveAccessibleDescription(
        messages.rules.name,
      );
    });
    expect(field(messages.labels.price)).toHaveAccessibleDescription(
      "El precio debe ser mayor que 0, con 2 decimales como máximo",
    );
    expect(field(messages.labels.name)).toHaveValue("Té verde");
    expect(field(messages.labels.name)).toHaveFocus();
  });

  it("shows a duplicate SKU on the SKU field", async () => {
    capturePost(() =>
      errorResponse(409, "SKU_ALREADY_EXISTS", [
        { field: "sku", message: "already exists" },
      ]),
    );
    const { user } = renderRoute("/products/new");

    await fillValidForm(user);
    await user.click(submit());

    await waitFor(() => {
      expect(field(messages.labels.sku)).toHaveAccessibleDescription(
        messages.feedback.skuTaken,
      );
    });
    expect(field(messages.labels.sku)).toHaveValue("  abc-123 ");
  });

  it.each([
    [500, "INTERNAL_ERROR", messages.feedback.unexpected],
    [429, "RATE_LIMITED", messages.feedback.rateLimited],
  ] as const)("keeps the values on a %i", async (status, code, text) => {
    capturePost(() => errorResponse(status, code, [], { "Retry-After": "1" }));
    const { user, router } = renderRoute("/products/new");

    await fillValidForm(user);
    await user.click(submit());

    expect(await screen.findByText(text)).toBeInTheDocument();
    expect(router.state.location.pathname).toBe("/products/new");
    expect(field(messages.labels.name)).toHaveValue("Té verde");
    expect(field(messages.labels.price)).toHaveValue("1,299.90");
  });

  it("disables the submit button while the request is pending", async () => {
    capturePost(async () => {
      await delay("infinite");
      return HttpResponse.json(created, { status: 201 });
    });
    const { user } = renderRoute("/products/new");

    await fillValidForm(user);
    await user.click(submit());

    await waitFor(() => {
      expect(submit()).toBeDisabled();
    });
  });

  it("adds thousands separators and ignores a typed comma with a hint", async () => {
    const { user, container } = renderRoute("/products/new");
    const price = field(messages.labels.price);

    await user.type(price, "1299.9");
    expect(price).toHaveValue("1,299.9");

    await user.clear(price);
    await user.type(price, "199,90");

    expect(price).toHaveValue("19,990");
    expect(
      within(container).getByText(messages.hints.comma),
    ).toBeInTheDocument();
    expect(await axe(container)).toHaveNoViolations();
  });
});
