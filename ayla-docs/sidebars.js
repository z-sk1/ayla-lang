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
    "modules",
    {
      type: "category",
      label: "Features",
      collapsed: false,
      items: [
        "language/variables",
        "language/booleans",
        "language/strings",
        "language/lifetimes",
        {
          type: "category",
          label: "Type System",
          collapsed: false,
          items: [
            "language/types/types",
            "language/types/custom-types",
            "language/types/aliases",
            "language/types/boundaries",
            "language/types/interfaces",
          ],
        },
        {
          type: "category",
          label: "Control Flow",
          collapsed: false,
          items: [
            "language/control-flow/if",
            "language/control-flow/switch-case",
            "language/control-flow/for",
            "language/control-flow/while",
            "language/control-flow/break-continue",
            "language/control-flow/with",
          ],
        },
        "language/functions",
        {
          type: "category",
          label: "Data Structures",
          collapsed: false,
          items: [
            "language/data-structures/arrays",
            "language/data-structures/slices",
            "language/data-structures/enums",
            "language/data-structures/structs",
          ],
        },
        "language/methods",
        "language/pointers",
        "language/concurrency",
      ],
    },
  ],
};

export default sidebars;
