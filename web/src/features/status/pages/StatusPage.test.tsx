import { screen, waitFor } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";
import { axe } from "../../../test/axe";
import { apiUrl } from "../../../test/handlers";
import { renderWithProviders } from "../../../test/render";
import { server } from "../../../test/server";
import { StatusPage } from "./StatusPage";

function service(name: "API" | "Base de datos") {
  return screen.getByRole("region", { name });
}

describe("StatusPage", () => {
  it("shows a loading state while checking", async () => {
    server.use(
      http.get(apiUrl("/healthz"), () => delay("infinite")),
      http.get(apiUrl("/readyz"), () => delay("infinite")),
    );
    const { container } = renderWithProviders(<StatusPage />);

    expect(screen.getByRole("status")).toHaveTextContent(
      "Comprobando el estado…",
    );
    expect(await axe(container)).toHaveNoViolations();
  });

  it("shows the API and the database as operational", async () => {
    const { container } = renderWithProviders(<StatusPage />);

    expect(
      await screen.findByRole("region", { name: "API" }),
    ).toHaveTextContent("En funcionamiento");
    expect(service("Base de datos")).toHaveTextContent("En funcionamiento");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(await axe(container)).toHaveNoViolations();
  });

  it("shows the database as unavailable when the API reports it", async () => {
    server.use(
      http.get(apiUrl("/readyz"), () =>
        HttpResponse.json(
          { status: "unavailable", checks: { database: "unavailable" } },
          { status: 503 },
        ),
      ),
    );
    const { container } = renderWithProviders(<StatusPage />);

    expect(
      await screen.findByRole("region", { name: "API" }),
    ).toHaveTextContent("En funcionamiento");
    expect(service("Base de datos")).toHaveTextContent("No disponible");
    expect(await axe(container)).toHaveNoViolations();
  });

  it("shows the API as unavailable and recovers on retry", async () => {
    server.use(
      http.get(apiUrl("/healthz"), () => HttpResponse.error()),
      http.get(apiUrl("/readyz"), () => HttpResponse.error()),
    );
    const { user, container } = renderWithProviders(<StatusPage />);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "La API no responde.",
    );
    expect(service("API")).toHaveTextContent("No disponible");
    expect(service("Base de datos")).toHaveTextContent("Desconocido");
    expect(await axe(container)).toHaveNoViolations();

    let release = () => {};
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    server.use(
      http.get(apiUrl("/healthz"), async () => {
        await gate;
        return HttpResponse.json({ status: "ok" });
      }),
      http.get(apiUrl("/readyz"), async () => {
        await gate;
        return HttpResponse.json({ status: "ok", checks: { database: "ok" } });
      }),
    );
    const retry = screen.getByRole("button", { name: "Reintentar" });

    await user.click(retry);

    await waitFor(() => {
      expect(retry).toBeDisabled();
    });
    release();
    await waitFor(() => {
      expect(service("API")).toHaveTextContent("En funcionamiento");
    });
    expect(service("Base de datos")).toHaveTextContent("En funcionamiento");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});
