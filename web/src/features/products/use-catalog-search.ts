import { useDebouncedCallback } from "@mantine/hooks";
import { useMemo, useRef } from "react";
import { useSearchParams } from "react-router";
import {
  defaultCatalogSearch,
  parseCatalogSearch,
  serializeCatalogSearch,
  type CatalogSearch,
} from "./search-params";

const debounceMs = 300;

type Filters = Partial<Omit<CatalogSearch, "page">>;

/**
 * The catalog view in the URL (R18). Filter, sort and page changes add a
 * history entry, and filter and sort changes go back to page 1. The search
 * applies 300 ms after the last keystroke or on Enter; the first update of a
 * typing session adds an entry and the rest replace it, so Back returns to
 * the view before the search.
 */
export function useCatalogSearch() {
  const [params, setParams] = useSearchParams();
  const search = useMemo(() => parseCatalogSearch(params), [params]);
  const typingSession = useRef(false);

  function navigate(next: CatalogSearch, replace = false) {
    setParams(serializeCatalogSearch(next), { replace });
  }

  function setFilters(filters: Filters) {
    typingSession.current = false;
    navigate({ ...search, ...filters, page: 1 });
  }

  function applySearch(text: string) {
    const q = text.trim() || undefined;
    if (q === search.q) {
      return;
    }
    navigate({ ...search, q, page: 1 }, typingSession.current);
    typingSession.current = true;
  }

  const searchLater = useDebouncedCallback(applySearch, debounceMs);
  const setPriceRangeLater = useDebouncedCallback(
    (minPrice: number | undefined, maxPrice: number | undefined) => {
      if (minPrice !== search.minPrice || maxPrice !== search.maxPrice) {
        setFilters({ minPrice, maxPrice });
      }
    },
    debounceMs,
  );

  return {
    search,
    setFilters,
    setPage: (page: number) => {
      navigate({ ...search, page });
    },
    /** Applies the search after a pause in typing. */
    searchLater,
    /** Applies the search at once (Enter) and ends the typing session. */
    searchNow: (text: string) => {
      searchLater.cancel();
      applySearch(text);
      typingSession.current = false;
    },
    endTypingSession: () => {
      typingSession.current = false;
    },
    setPriceRangeLater,
    cancelPriceRange: () => {
      setPriceRangeLater.cancel();
    },
    clearFilters: () => {
      searchLater.cancel();
      setPriceRangeLater.cancel();
      typingSession.current = false;
      navigate({
        ...defaultCatalogSearch,
        sort: search.sort,
        order: search.order,
      });
    },
  };
}

export type CatalogSearchControls = ReturnType<typeof useCatalogSearch>;
