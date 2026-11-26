import { MantineProvider } from "@mantine/core";
import { ModalsProvider } from "@mantine/modals";
import { Notifications } from "@mantine/notifications";
import type { ReactNode } from "react";
import { cssVariablesResolver, theme } from "./theme";

interface UiProviderProps {
  children: ReactNode;
  /** `"test"` disables transitions and portals (Mantine's Vitest guide). */
  env?: "default" | "test";
}

/** Mantine with the project theme in light mode, notifications and modals. */
export function UiProvider({ children, env }: UiProviderProps) {
  return (
    <MantineProvider
      theme={theme}
      cssVariablesResolver={cssVariablesResolver}
      forceColorScheme="light"
      env={env}
    >
      <Notifications />
      <ModalsProvider>{children}</ModalsProvider>
    </MantineProvider>
  );
}
