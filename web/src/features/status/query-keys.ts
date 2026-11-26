export const statusKeys = {
  all: ["status"] as const,
  system: () => [...statusKeys.all, "system"] as const,
};
