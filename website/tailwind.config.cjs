/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{astro,html,js,jsx,md,mdx,ts,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        paper: "#FAF9F6",
        ink: "#0E0E10",
        rule: "#1f1f24",
        // research-ink-blue accent
        accent: {
          DEFAULT: "#1E3A8A",
          soft: "#3b58a8",
        },
        // signal orange — used ONLY for "noise" / privacy callouts
        noise: {
          DEFAULT: "#E25822",
          soft: "#f37e4d",
        },
        // dark-mode surfaces
        nightInk: "#F5F4F0",
        nightPaper: "#0E0E10",
        nightRule: "#26262b",
      },
      fontFamily: {
        display: [
          "Satoshi",
          '"Helvetica Neue"',
          "Helvetica",
          "Arial",
          "sans-serif",
        ],
        sans: [
          "Satoshi",
          '"Helvetica Neue"',
          "Helvetica",
          "Arial",
          "sans-serif",
        ],
        mono: [
          '"JetBrains Mono"',
          "ui-monospace",
          "SFMono-Regular",
          "Menlo",
          "monospace",
        ],
      },
      letterSpacing: {
        tightish: "-0.012em",
        tightest: "-0.025em",
      },
      maxWidth: {
        prose: "44rem", // 704px
        wide: "76rem",
      },
      fontSize: {
        // research-paper scale — body is generous, display is dramatic
        "display-1": ["clamp(3rem, 7vw, 5.75rem)", { lineHeight: "0.96", letterSpacing: "-0.02em" }],
        "display-2": ["clamp(2rem, 4.5vw, 3.25rem)", { lineHeight: "1.04", letterSpacing: "-0.018em" }],
        "h2": ["clamp(1.5rem, 2.4vw, 2rem)", { lineHeight: "1.15", letterSpacing: "-0.012em" }],
      },
    },
  },
  plugins: [],
};
