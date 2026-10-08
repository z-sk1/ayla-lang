import { useEffect, useState } from "react";
import Editor from "@monaco-editor/react";

import { initWasm, runAyla } from "./wasm";
import { registerAyla } from "./ayla";

import "./playground.css";

export default function Playground() {
  const [code, setCode] = useState(`putln("Hello, world!")`);
  const [output, setOutput] = useState("Loading...");
  const [ready, setReady] = useState(false);

  useEffect(() => {
    async function load() {
      try {
        await initWasm();
        setReady(true);
        setOutput("");
      } catch (err) {
        setOutput(String(err));
      }
    }

    load();
  }, []);

  async function run() {
    if (!ready) return;

    try {
      setOutput("Running...");
      const result = await runAyla(code);
      setOutput(result ?? "");
    } catch (err) {
      setOutput(String(err));
    }
  }

  return (
    <div className="playground">
      <div className="editor-panel">
        <div className="toolbar">
          <h2>Ayla Playground</h2>

          <button disabled={!ready} onClick={run}>
            ▶ Run
          </button>
        </div>

        <Editor
          height="80vh"
          language="ayla"
          theme="ayla-dark"
          value={code}
          beforeMount={(monaco) => registerAyla(monaco)}
          onChange={(value) => setCode(value ?? "")}
          options={{
            automaticLayout: true,
            minimap: {
              enabled: false,
            },
            fontSize: 15,
            fontLigatures: true,
            tabSize: 4,
            scrollBeyondLastLine: false,
            roundedSelection: true,
            smoothScrolling: true,
            padding: {
              top: 12,
              bottom: 12,
            },
          }}
        />
      </div>

      <div className="output-panel">
        <h3>Output</h3>

        <pre>{output}</pre>
      </div>
    </div>
  );
}
