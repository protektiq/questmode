const app = document.querySelector<HTMLDivElement>("#app");
if (!app) {
  throw new Error("Missing #app root element");
}

const escapeHtml = (value: string): string =>
  value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");

const setStatus = (html: string) => {
  app.innerHTML = html;
};

setStatus(`<main style="font-family: system-ui, sans-serif; max-width: 40rem; margin: 2rem auto; padding: 0 1rem;">
  <h1 style="font-size: 1.5rem;">Quest Mode</h1>
  <p role="status" aria-live="polite">Checking API…</p>
</main>`);

void (async () => {
  try {
    const res = await fetch("/api/health");
    const body = await res.text();
    const statusLine = res.ok
      ? `OK (${res.status})`
      : `Error (${res.status})`;
    setStatus(`<main style="font-family: system-ui, sans-serif; max-width: 40rem; margin: 2rem auto; padding: 0 1rem;">
  <h1 style="font-size: 1.5rem;">Quest Mode</h1>
  <p role="status" aria-live="polite">${escapeHtml(statusLine)}</p>
  <pre style="white-space: pre-wrap; word-break: break-word; background: #f4f4f5; padding: 1rem; border-radius: 0.5rem;">${escapeHtml(body)}</pre>
</main>`);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    setStatus(`<main style="font-family: system-ui, sans-serif; max-width: 40rem; margin: 2rem auto; padding: 0 1rem;">
  <h1 style="font-size: 1.5rem;">Quest Mode</h1>
  <p role="status" aria-live="assertive">Request failed</p>
  <pre style="white-space: pre-wrap; word-break: break-word; background: #fef2f2; padding: 1rem; border-radius: 0.5rem;">${escapeHtml(message)}</pre>
</main>`);
  }
})();
