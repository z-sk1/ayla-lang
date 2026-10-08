import React, { useEffect, useRef, useState } from "react";

export default function Playground() {
  const [code, setCode] = useState(`println("Hello, world!")`);
  const [output, setOutput] = useState("");

  useEffect(() => {
    const load = async () => {
      const script = document.createElement("script");
      script.src = "/wasm_exec.js";

      script.onload = async () => {
        const go = new Go();

        const result = await WebAssembly.instantiateStreaming(
          fetch("/ayla.wasm"),
          go.importObject
        );

        go.run(result.instance);
      };

      document.body.appendChild(script);
    };

    load();
  }, []);

  const run = () => {
    try {
      // @ts-ignore
      setOutput(window.runAyla(code));
    } catch (e) {
      setOutput(String(e));
    }
  };

  return (
    <main style={{ padding: 24 }}>
      <h1>Ayla Playground</h1>

      <textarea
        value={code}
        onChange={e => setCode(e.target.value)}
        rows={20}
        style={{
          width: "100%",
          fontFamily: "monospace",
          fontSize: 15
        }}
      />

      <button onClick={run}>Run</button>

      <pre>{output}</pre>
    </main>
  );
}
