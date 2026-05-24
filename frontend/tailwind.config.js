/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{svelte,js,ts,jsx,tsx}",
  ],
  darkMode: "class",
  theme: {
    extend: {
      "colors": {
        "on-primary-fixed-variant": "#7b2f12",
        "surface-card": "#FFFFFF",
        "secondary-fixed": "#e5e2df",
        "on-tertiary-fixed": "#00201c",
        "inverse-primary": "#ffb59c",
        "tertiary-fixed-dim": "#59daca",
        "primary-fixed": "#ffdbd0",
        "on-surface": "#221a17",
        "primary-container": "#e37e5b",
        "surface-tint": "#994527",
        "on-secondary": "#ffffff",
        "on-tertiary-container": "#003933",
        "on-surface-variant": "#55433d",
        "surface-container-high": "#f6e4df",
        "secondary-fixed-dim": "#c8c6c3",
        "tertiary": "#006a61",
        "surface-container-lowest": "#ffffff",
        "primary": "#994527",
        "on-error-container": "#93000a",
        "background": "#fff8f6",
        "background-page": "#F0F0F0",
        "on-primary-container": "#5e1a00",
        "secondary": "#5f5e5c",
        "surface-container-low": "#fff1ed",
        "primary-fixed-dim": "#ffb59c",
        "surface-variant": "#f0dfda",
        "surface": "#fff8f6",
        "secondary-container": "#e2dfdc",
        "on-tertiary-fixed-variant": "#005048",
        "text-main": "#1A1A1A",
        "outline-variant": "#dbc1b9",
        "on-tertiary": "#ffffff",
        "status-active": "#6BBF9E",
        "inverse-surface": "#382e2b",
        "on-primary-fixed": "#390c00",
        "status-warning": "#EBA059",
        "outline": "#88726b",
        "error-container": "#ffdad6",
        "surface-container": "#fceae5",
        "on-secondary-fixed": "#1c1c1a",
        "inverse-on-surface": "#ffede8",
        "on-secondary-container": "#636260",
        "on-secondary-fixed-variant": "#474745",
        "text-muted": "#757575",
        "on-primary": "#ffffff",
        "surface-container-highest": "#f0dfda",
        "surface-dim": "#e8d6d1",
        "tertiary-container": "#0fac9d",
        "tertiary-fixed": "#79f7e6",
        "error": "#ba1a1a",
        "on-background": "#221a17",
        "on-error": "#ffffff",
        "surface-bright": "#fff8f6"
      },
      "borderRadius": {
        "DEFAULT": "0.25rem",
        "lg": "0.5rem",
        "xl": "0.75rem",
        "full": "9999px",
        "2xl": "1.5rem",
        "3xl": "2rem"
      },
      "spacing": {
        "card-padding": "24px",
        "base": "8px",
        "margin-page": "32px",
        "gutter": "24px",
        "stack-gap": "16px"
      },
      "fontFamily": {
        "label-sm": ["Plus Jakarta Sans"],
        "body-md": ["Plus Jakarta Sans"],
        "headline-md": ["Plus Jakarta Sans"],
        "body-lg": ["Plus Jakarta Sans"],
        "headline-md-mobile": ["Plus Jakarta Sans"],
        "display-lg": ["Plus Jakarta Sans"]
      },
      "fontSize": {
        "label-sm": ["12px", { "lineHeight": "16px", "letterSpacing": "0.05em", "fontWeight": "600" }],
        "body-md": ["16px", { "lineHeight": "24px", "fontWeight": "400" }],
        "headline-md": ["24px", { "lineHeight": "32px", "fontWeight": "600" }],
        "body-lg": ["18px", { "lineHeight": "28px", "fontWeight": "400" }],
        "headline-md-mobile": ["20px", { "lineHeight": "28px", "fontWeight": "600" }],
        "display-lg": ["40px", { "lineHeight": "48px", "letterSpacing": "-0.02em", "fontWeight": "700" }]
      },
      "boxShadow": {
        "ambient-low": "0 4px 12px rgba(0, 0, 0, 0.03)",
        "ambient-hover": "0 8px 24px rgba(0, 0, 0, 0.05)",
        "ambient-card": "0 4px 12px rgba(0, 0, 0, 0.03)",
        "ambient-button": "0 2px 8px rgba(0, 0, 0, 0.08)"
      }
    }
  }
}
