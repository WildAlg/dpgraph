// @ts-check
import { defineConfig } from "astro/config";
import mdx from "@astrojs/mdx";
import react from "@astrojs/react";
import sitemap from "@astrojs/sitemap";
import tailwind from "@astrojs/tailwind";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";

const SITE = "https://wildalg.github.io";
const BASE = "/dpgraph";

export default defineConfig({
  site: SITE,
  base: BASE,
  trailingSlash: "ignore",
  integrations: [
    tailwind({ applyBaseStyles: false }),
    react(),
    mdx(),
    sitemap(),
  ],
  markdown: {
    remarkPlugins: [remarkMath],
    rehypePlugins: [rehypeKatex],
    shikiConfig: {
      themes: { light: "github-light", dark: "github-dark-default" },
      wrap: true,
    },
  },
  build: {
    assets: "_assets",
  },
});
