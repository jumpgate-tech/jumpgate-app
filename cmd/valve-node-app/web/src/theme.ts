// Global light/dark/system theme. The whole app is tokenized (style.css's
// :root, panel.css's .p-wrap), all dark by default; the light theme is a set
// of token overrides gated on <html data-theme="light">. This module owns the
// one piece of runtime state that drives it: the operator's preference
// (System / Light / Dark), persisted in localStorage, resolved to a concrete
// "light" | "dark" that it stamps on the document element.
//
// "System" is resolved HERE via matchMedia, not in CSS, so the stylesheet only
// ever needs the single [data-theme="light"] override branch (never a
// prefers-color-scheme copy of it), and an OS appearance change while the app
// is open re-resolves live.
export type ThemePref = "system" | "light" | "dark";

const KEY = "valve-theme";
const media = window.matchMedia("(prefers-color-scheme: dark)");

export function getThemePref(): ThemePref {
  const v = localStorage.getItem(KEY);
  return v === "light" || v === "dark" || v === "system" ? v : "system";
}

// resolveTheme collapses the preference to the concrete theme the DOM wears:
// "system" follows the OS via matchMedia; the two explicit prefs pass through.
export function resolveTheme(pref: ThemePref = getThemePref()): "light" | "dark" {
  if (pref === "system") return media.matches ? "dark" : "light";
  return pref;
}

function apply(): void {
  document.documentElement.dataset.theme = resolveTheme();
}

export function setThemePref(pref: ThemePref): void {
  localStorage.setItem(KEY, pref);
  apply();
}

// initTheme stamps the resolved preference onto <html> as early as possible and
// keeps a "system" preference tracking the OS live (a no-op re-apply when the
// pref is explicit). Call once, before the first screen renders.
export function initTheme(): void {
  apply();
  media.addEventListener("change", () => {
    if (getThemePref() === "system") apply();
  });
}
