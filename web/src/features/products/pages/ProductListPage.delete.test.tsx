import { screen, waitFor, within } from "@testing-library/react";
import { HttpResponse } from "msw";
import { describe, expect, it } from "vitest";
import { axe } from "../../../test/axe";
import { renderRoute } from "../../../test/render";
import { server } from "../../../test/server";
import { messages } from "../messages";
import { productKeys } from "../query-keys";
import {
  buildList,
  buildProduct,
  errorResponse,
  productHandlers,
} from "../testing";
import type { Product } from "../types";

const cafe = buildProduct({
  id: "0193a5b0-7c1e-7000-8000-0000000000c1",
  sku: "CAF-0123",
  name: "Café molido",
  version: 4,
});
const silla = buildProduct({
  id: "0193a5b0-7c1e-7000-8000-0000000000c2",
  sku: "SIL-0001",
  name: "Silla plegable",
});

/** A catalog whose list reflects deletions the API accepted. */
function serveCatalog(
  products: Product[],
  deleteResponse: () => Response | Promise<Response>,
) {
  let current = products;
  const deletes: { id: string; version: string | null }[] = [];
  server.use(
    productHandlers.list(() => HttpResponse.json(buildList(current))),
    productHandlers.remove(async ({ request, params }) => {
      const id = String(params.id);
      deletes.push({
        id,
        version: new URL(request.url).searchParams.get("version"),
      });
      const response = await deleteResponse();
      if (response.status === 204 || response.status === 404) {
        current = current.filter((p) => p.id !== id);
      }
      return response;
    }),
  );
  return deletes;
}

function deleteButton(name: string) {
  return screen.getByRole("button", {
    name: `${messages.actions.delete} ${name}`,
  });
}

function dialog() {
  return screen.getByRole("dialog", { name: messages.deleteConfirm.title });
}

function confirmButton() {
  return within(dialog()).getByRole("button", {
    name: messages.actions.delete,
  });
}

/** A response that waits until `release` is called. */
function gated(response: () => Response) {
  let release = () => {};
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });
  return {
    release,
    respond: async () => {
      await gate;
      return response();
    },
  };
}

describe("ProductListPage: delete", () => {
  it("asks for confirmation naming the product, and cancel changes nothing", async () => {
    const deletes = serveCatalog(
      [cafe, silla],
      () => new HttpResponse(null, { status: 204 }),
    );
    const { user, container } = renderRoute("/products");
    await screen.findByRole("article", { name: "Café molido" });

    await user.click(deleteButton("Café molido"));

    expect(dialog()).toHaveTextContent(
      "Se eliminará «Café molido» (CAF-0123). Esta acción no se puede deshacer.",
    );
    expect(await axe(container)).toHaveNoViolations();

    await user.click(
      within(dialog()).getByRole("button", { name: messages.actions.cancel }),
    );

    await waitFor(() => {
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    });
    expect(deletes).toEqual([]);
    expect(
      screen.getByRole("article", { name: "Café molido" }),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(deleteButton("Café molido")).toHaveFocus();
    });
  });

  it("removes the row at once, then confirms, sending the listed version", async () => {
    const gate = gated(() => new HttpResponse(null, { status: 204 }));
    const deletes = serveCatalog([cafe, silla], gate.respond);
    const { user } = renderRoute("/products");
    await screen.findByRole("article", { name: "Café molido" });

    await user.click(deleteButton("Café molido"));
    await user.click(confirmButton());

    await waitFor(() => {
      expect(
        screen.queryByRole("article", { name: "Café molido" }),
      ).not.toBeInTheDocument();
    });
    expect(confirmButton()).toBeDisabled();
    expect(deletes).toEqual([{ id: cafe.id, version: "4" }]);

    gate.release();

    expect(
      await screen.findByText(messages.feedback.deleted),
    ).toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(
      screen.getByRole("article", { name: "Silla plegable" }),
    ).toBeInTheDocument();
  });

  it("brings the row back with a notice when the deletion fails", async () => {
    serveCatalog([cafe, silla], () => errorResponse(500, "INTERNAL_ERROR"));
    const { user } = renderRoute("/products");
    await screen.findByRole("article", { name: "Café molido" });

    await user.click(deleteButton("Café molido"));
    await user.click(confirmButton());

    expect(
      await screen.findByText(messages.feedback.deleteFailed),
    ).toBeInTheDocument();
    expect(
      await screen.findByRole("article", { name: "Café molido" }),
    ).toBeInTheDocument();
  });

  it("keeps the row and warns about a concurrent change", async () => {
    serveCatalog([cafe, silla], () => errorResponse(409, "VERSION_CONFLICT"));
    const { user, container } = renderRoute("/products");
    await screen.findByRole("article", { name: "Café molido" });

    await user.click(deleteButton("Café molido"));
    await user.click(confirmButton());

    expect(
      await screen.findByText(messages.feedback.conflict),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("article", { name: "Café molido" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: messages.actions.reload }),
    ).toBeInTheDocument();
    expect(await axe(container)).toHaveNoViolations();
  });

  it("hides the conflict notice once the list reloads", async () => {
    serveCatalog([cafe, silla], () => errorResponse(409, "VERSION_CONFLICT"));
    const { user } = renderRoute("/products");
    await screen.findByRole("article", { name: "Café molido" });

    await user.click(deleteButton("Café molido"));
    await user.click(confirmButton());
    expect(
      await screen.findByText(messages.feedback.conflict),
    ).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: messages.actions.reload }),
    );

    await waitFor(() => {
      expect(
        screen.queryByText(messages.feedback.conflict),
      ).not.toBeInTheDocument();
    });
  });

  it("hides the conflict notice after an automatic refetch", async () => {
    serveCatalog([cafe, silla], () => errorResponse(409, "VERSION_CONFLICT"));
    const { user, queryClient } = renderRoute("/products");
    await screen.findByRole("article", { name: "Café molido" });

    await user.click(deleteButton("Café molido"));
    await user.click(confirmButton());
    expect(
      await screen.findByText(messages.feedback.conflict),
    ).toBeInTheDocument();

    await queryClient.refetchQueries({ queryKey: productKeys.lists() });

    await waitFor(() => {
      expect(
        screen.queryByText(messages.feedback.conflict),
      ).not.toBeInTheDocument();
    });
  });

  it("hides the conflict notice after another successful deletion", async () => {
    let calls = 0;
    serveCatalog([cafe, silla], () => {
      calls += 1;
      return calls === 1
        ? errorResponse(409, "VERSION_CONFLICT")
        : new HttpResponse(null, { status: 204 });
    });
    const { user } = renderRoute("/products");
    await screen.findByRole("article", { name: "Café molido" });

    await user.click(deleteButton("Café molido"));
    await user.click(confirmButton());
    expect(
      await screen.findByText(messages.feedback.conflict),
    ).toBeInTheDocument();

    await user.click(deleteButton("Silla plegable"));
    await user.click(confirmButton());

    expect(
      await screen.findByText(messages.feedback.deleted),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(
        screen.queryByText(messages.feedback.conflict),
      ).not.toBeInTheDocument();
    });
  });

  it("removes a product someone else already deleted, with a notice", async () => {
    serveCatalog([cafe, silla], () => errorResponse(404, "NOT_FOUND"));
    const { user } = renderRoute("/products");
    await screen.findByRole("article", { name: "Café molido" });

    await user.click(deleteButton("Café molido"));
    await user.click(confirmButton());

    expect(
      await screen.findByText(messages.feedback.alreadyDeleted),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("article", { name: "Café molido" }),
    ).not.toBeInTheDocument();
  });

  it("shows the previous page after deleting the only product on the last one", async () => {
    let total = 21;
    server.use(
      productHandlers.list(({ request }) => {
        const page = Number(new URL(request.url).searchParams.get("page"));
        const data = page === 2 ? (total === 21 ? [silla] : []) : [cafe];
        return HttpResponse.json(buildList(data, { total, page }));
      }),
      productHandlers.remove(() => {
        total = 20;
        return new HttpResponse(null, { status: 204 });
      }),
    );
    const { user, router } = renderRoute("/products?page=2");
    await screen.findByRole("article", { name: "Silla plegable" });

    await user.click(deleteButton("Silla plegable"));
    await user.click(confirmButton());

    await waitFor(() => {
      expect(router.state.location.search).toBe("");
    });
    expect(
      await screen.findByRole("article", { name: "Café molido" }),
    ).toBeInTheDocument();
  });
});
