import {
  createTheme,
  type CSSVariablesResolver,
  type MantineColorsTuple,
} from "@mantine/core";

// Generado con @mantine/colors-generator desde #4c5fd5
const brand: MantineColorsTuple = [
  "#ecf0ff",
  "#d7dcfb",
  "#aeb7ef",
  "#818fe3",
  "#5c6dd9",
  "#4c5fd5",
  "#374dd1",
  "#283eba",
  "#2137a7",
  "#142f95",
];

export const theme = createTheme({
  colors: { brand },
  primaryColor: "brand",
  primaryShade: 6,
  black: "#14161a",
  white: "#ffffff",
  defaultRadius: "md",
});

export const cssVariablesResolver: CSSVariablesResolver = () => ({
  variables: {},
  light: {
    // Texto "dimmed" con gray.7 para cumplir WCAG AA (gray.6 da 3.3:1 sobre blanco)
    "--mantine-color-dimmed": "var(--mantine-color-gray-7)",
    // Bordes por defecto (variant="default", withBorder) en gray.3; Mantine usa gray.4
    "--mantine-color-default-border": "var(--mantine-color-gray-3)",
  },
  dark: {},
});
