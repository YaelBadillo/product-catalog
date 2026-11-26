import { Button } from "@mantine/core";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { axe } from "../../test/axe";
import { renderWithProviders } from "../../test/render";
import { PageState } from "./PageState";

describe("PageState", () => {
  it("renders the title, the text and the action", async () => {
    const { container } = renderWithProviders(
      <PageState
        title="Aún no hay productos"
        text="Registra el primero."
        action={<Button>Registrar producto</Button>}
      />,
    );

    expect(
      screen.getByRole("heading", { level: 2, name: "Aún no hay productos" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Registra el primero.")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Registrar producto" }),
    ).toBeInTheDocument();
    expect(await axe(container)).toHaveNoViolations();
  });

  it("renders only the title, at the requested level", async () => {
    const { container } = renderWithProviders(
      <PageState title="Producto no encontrado" headingLevel={1} />,
    );

    expect(
      screen.getByRole("heading", { level: 1, name: "Producto no encontrado" }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(await axe(container)).toHaveNoViolations();
  });
});
