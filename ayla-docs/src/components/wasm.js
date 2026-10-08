let ready = false;
let loading = null;

export async function initWasm() {
  if (ready) return;
  if (loading) return loading;

  const base = "/ayla-lang/";

  loading = (async () => {
    await new Promise((resolve, reject) => {
      if (window.Go) return resolve();

      const script = document.createElement("script");
      script.src = base + "wasm_exec.js";
      script.onload = resolve;
      script.onerror = reject;
      document.body.appendChild(script);
    });

    const go = new window.Go();

    const result = await WebAssembly.instantiateStreaming(
      fetch(base + "ayla.wasm"),
      go.importObject
    );

    go.run(result.instance);
    ready = true;
  })();

  return loading;
}

export async function runAyla(code) {
  await initWasm();
  return window.runAyla(code);
}
