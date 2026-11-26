import { screen, waitFor, within } from "@testing-library/react";
import { delay, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";
import { renderRoute } from "../../../test/render";
import { server } from "../../../test/server";
import { messages } from "../messages";
import {
  buildList,
  buildProduct,
  errorResponse,
  productHandlers,
} from "../testing";

const product = buildProduct({
  id: "0193a5b0-7c1e-7000-8000-0000000000d1",
  name: "Café molido",
  sku: "CAF-0123",
  version: 2,
});
const path = `/products/${product.id}`;

function serveDetail(deleteResponse: () => Response | Promise<Response>) {
  server.use(
    productHandlers.getOf(product),
    productHandlers.listOf(buildList([])),
    productHandlers.remove(deleteResponse),
  );
}

async function confirmDelete(user: ReturnType<typeof renderRoute>["user"]) {
  await screen.findByRole("heading", { level: 1, name: "Café molido" });
  await user.click(
    screen.getByRole("button", { name: messages.actions.delete }),
  );
  const dialog = screen.getByRole("dialog", {
    name: messages.deleteConfirm.title,
  });
  expect(dialog).toHaveTextContent("«Café molido» (CAF-0123)");
  await user.click(
    within(dialog).getByRole("button", { name: messages.actions.delete }),
  );
}

describe("ProductDetailPage: delete", () => {
  it("goes to the catalog with a confirmation", async () => {
    serveDetail(() => new HttpResponse(null, { status: 204 }));
    const { user, router } = renderRoute(path);

    await confirmDelete(user);

    expect(
      await screen.findByText(messages.feedback.deleted),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/products");
    });
  });

  it("disables the delete action while the request is pending", async () => {
    serveDetail(async () => {
      await delay("infinite");
      return new HttpResponse(null, { status: 204 });
    });
    const { user } = renderRoute(path);

    await confirmDelete(user);

    await waitFor(() => {
      expect(
        within(screen.getByRole("main")).getByRole("button", {
          name: messages.actions.delete,
        }),
      ).toBeDisabled();
    });
  });

  it("stays with a notice when the deletion fails", async () => {
    serveDetail(() => errorResponse(500, "INTERNAL_ERROR"));
    const { user, router } = renderRoute(path);

    await confirmDelete(user);

    expect(
      await screen.findByText(messages.feedback.deleteFailed),
    ).toBeInTheDocument();
    expect(router.state.location.pathname).toBe(path);
    expect(
      screen.getByRole("heading", { level: 1, name: "Café molido" }),
    ).toBeInTheDocument();
  });

  it("warns about a concurrent change", async () => {
    serveDetail(() => errorResponse(409, "VERSION_CONFLICT"));
    const { user, router } = renderRoute(path);

    await confirmDelete(user);

    expect(
      await screen.findByText(messages.feedback.conflict),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: messages.actions.reload }),
    ).toBeInTheDocument();
    expect(router.state.location.pathname).toBe(path);
  });

  it("goes to the catalog when someone else already deleted it", async () => {
    serveDetail(() => errorResponse(404, "NOT_FOUND"));
    const { user, router } = renderRoute(path);

    await confirmDelete(user);

    expect(
      await screen.findByText(messages.feedback.alreadyDeleted),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/products");
    });
  });
});
