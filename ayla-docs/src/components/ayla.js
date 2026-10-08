let registered = false;

export function registerAyla(monaco) {
  if (registered) return;
  registered = true;

  monaco.languages.register({
    id: "ayla",
  });

  monaco.languages.setLanguageConfiguration("ayla", {
    comments: {
      lineComment: "//",
      blockComment: ["/*", "*/"],
    },
    autoClosingPairs: [
      { open: "{", close: "}" },
      { open: "(", close: ")" },
      { open: "[", close: "]" },
      { open: '"', close: '"' },
    ],
    surroundingPairs: [
      { open: "{", close: "}" },
      { open: "(", close: ")" },
      { open: "[", close: "]" },
      { open: '"', close: '"' },
    ],
  });

  monaco.languages.setMonarchTokensProvider("ayla", {
    keywords: [
      "ayla",
      "elen",
      "for",
      "while",
      "give",
      "snap",
      "next",
      "choose",
      "when",
      "otherwise",
      "start",
      "with",
      "it",
      "in",
      "range",
      "interface",
      "say",
      "keep",
      "import",
      "enum",
      "struct",
      "map",
      "type",
      "fun",
    ],

    typeKeywords: [
      "int",
      "float",
      "string",
      "bool",
      "thing",
      "error",
    ],

    tokenizer: {
      root: [
        [/\/\/.*$/, "comment"],
        [/\/\*/, "comment", "@comment"],

        [/"([^"\\]|\\.)*"/, "string"],
        [/\d+\.\d+/, "number.float"],
        [/\d+/, "number"],

        [/\b(yes|no|nil)\b/, "constant"],

        [
          /\b(struct)\s+([A-Z][A-Za-z0-9_]*)/,
          ["keyword", "type.identifier"],
        ],

        [
          /\b(fun)\s+([a-z_][A-Za-z0-9_]*)/,
          ["keyword", "function.definition"],
        ],

        [/\b[A-Z][A-Za-z0-9_]*\b/, "type.identifier"],

        [/\.[a-zA-Z_][A-Za-z0-9_]*/, "member"],

        [/\b[a-z_][A-Za-z0-9_]*(?=\()/, "function"],

        [
          /[a-zA-Z_][A-Za-z0-9_]*/,
          {
            cases: {
              "@keywords": "keyword",
              "@typeKeywords": "type",
              "@default": "identifier",
            },
          },
        ],

        [/[+\-*\/=!<>:&|]+/, "operator"],
        [/[{}()[\]]/, "@brackets"],
        [/[;,.]/, "delimiter"],
      ],

      comment: [
        [/[^\/*]+/, "comment"],
        [/\*\//, "comment", "@pop"],
        [/[\/*]/, "comment"],
      ],
    },
  });

  monaco.editor.defineTheme("ayla-dark", {
    base: "vs-dark",
    inherit: true,
    rules: [
      { token: "keyword", foreground: "C586C0", fontStyle: "bold" },
      { token: "type", foreground: "4EC9B0" },
      { token: "type.identifier", foreground: "4EC9B0" },
      { token: "function.definition", foreground: "DCDCAA", fontStyle: "bold" },
      { token: "function", foreground: "DCDCAA" },
      { token: "member", foreground: "9CDCFE" },
      { token: "string", foreground: "CE9178" },
      { token: "number", foreground: "B5CEA8" },
      { token: "number.float", foreground: "B5CEA8" },
      { token: "comment", foreground: "6A9955", fontStyle: "italic" },
      { token: "constant", foreground: "569CD6" },
      { token: "operator", foreground: "D4D4D4" },
    ],
    colors: {
      "editor.background": "#1E1E1E",
      "editor.foreground": "#D4D4D4",
      "editorLineNumber.foreground": "#858585",
      "editorCursor.foreground": "#AEAFAD",
      "editor.selectionBackground": "#264F78",
      "editor.inactiveSelectionBackground": "#3A3D41",
    },
  });
}
