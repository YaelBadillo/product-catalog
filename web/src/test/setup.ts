import "@testing-library/jest-dom/vitest";
import "vitest-axe/extend-expect";
import { notifications } from "@mantine/notifications";
import { cleanup } from "@testing-library/react";
import { afterAll, afterEach, beforeAll, vi } from "vitest";
import { server } from "./server";

// jsdom gaps that Mantine components hit (https://mantine.dev/guides/vitest/).
// Drop the pseudo-element argument, which jsdom does not implement.
const getComputedStyle = window.getComputedStyle.bind(window);
window.getComputedStyle = (element) => getComputedStyle(element);

window.HTMLElement.prototype.scrollIntoView = vi.fn();

// Mantine 9 listens with addEventListener only, so the deprecated
// addListener/removeListener are left out.
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }),
});

if (!("fonts" in document)) {
  Object.defineProperty(document, "fonts", {
    value: { addEventListener: vi.fn(), removeEventListener: vi.fn() },
  });
}

window.ResizeObserver = class {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
};

beforeAll(() => {
  server.listen({ onUnhandledFrame: "error" });
});

// RTL only auto-cleans up when `afterEach` is a global; Vitest globals are off.
afterEach(() => {
  cleanup();
  // Mantine keeps notifications in a global store.
  notifications.clean();
  server.resetHandlers();
});

afterAll(() => {
  server.close();
});
