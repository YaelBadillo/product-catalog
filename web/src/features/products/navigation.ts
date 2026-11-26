import { useLocation, useNavigate } from "react-router";

/** History state the catalog's links put on the detail and edit entries. */
export interface CatalogState {
  /** The previous history entry is the catalog. */
  fromCatalog?: boolean;
  /** The catalog query to restore, e.g. "?category=food". */
  catalogSearch?: string;
}

function readState(state: unknown): CatalogState {
  if (typeof state !== "object" || state === null) {
    return {};
  }
  const { fromCatalog, catalogSearch } = state as Record<string, unknown>;
  return {
    fromCatalog: fromCatalog === true,
    catalogSearch:
      typeof catalogSearch === "string" ? catalogSearch : undefined,
  };
}

/** The catalog state of the current history entry. */
export function useCatalogState(): CatalogState {
  return readState(useLocation().state);
}

/**
 * "Volver al catálogo": goes back when the previous entry is the catalog, so
 * its filters, search, sort and page return; otherwise opens the catalog with
 * the last known query, or the default view (contracts/web-routes.md).
 */
export function useBackToCatalog(): () => void {
  const navigate = useNavigate();
  const { fromCatalog, catalogSearch } = useCatalogState();
  return () => {
    if (fromCatalog) {
      void navigate(-1);
      return;
    }
    void navigate(`/products${catalogSearch ?? ""}`);
  };
}
