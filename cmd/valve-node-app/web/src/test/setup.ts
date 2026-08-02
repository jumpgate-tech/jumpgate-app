import "@testing-library/jest-dom/vitest";

// jsdom has no matchMedia; theme.ts touches it at import time (to resolve the
// "system" appearance), so every test that transitively imports a themed
// component needs this stub.
if (!window.matchMedia) {
  window.matchMedia = (query: string): MediaQueryList =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }) as unknown as MediaQueryList;
}
