// @ts-check

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.

 @type {import('@docusaurus/plugin-content-docs').SidebarsConfig}
 */
// sidebars.js

const sidebars = {
  tutorialSidebar: [
    "intro",
    "installation",
    "first-program",

    {
      type: "category",
      label: "Features",
      collapsed: false,
      items: [
        // "language/variables",
        // "language/types",
        // "language/control-flow",
        "language/functions",
      ],
    },

    // {
    //   type: "category",
    //   label: "Built-in Functions",
    //   collapsed: false,
    //   items: ["builtins/print", "builtins/len", "builtins/input"],
    // },
  ],
};

export default sidebars;
