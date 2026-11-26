import { useEffect } from "react";

const appName = "Catálogo de productos";

/** Sets `document.title` to "{title} · Catálogo de productos". */
export function useDocumentTitle(title: string): void {
  useEffect(() => {
    document.title = `${title} · ${appName}`;
  }, [title]);
}
