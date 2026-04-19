const app = document.querySelector<HTMLDivElement>("#app");
if (!app) {
  throw new Error("Missing #app root element");
}

app.innerHTML = `
  <main style="font-family: system-ui, sans-serif; max-width: 40rem; margin: 2rem auto; padding: 0 1rem;">
    <h1 style="font-size: 1.5rem;">Quest Mode</h1>
    <p>Vite + TypeScript frontend scaffold.</p>
  </main>
`;
