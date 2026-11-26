import { clsx } from "clsx";
import { CircleCheck, CircleHelp, CircleX } from "lucide-react";
import { useId } from "react";
import type { ServiceState } from "../api";

const presentation = {
  operational: {
    label: "Operational",
    Icon: CircleCheck,
    color: "text-green-700",
  },
  unavailable: { label: "Unavailable", Icon: CircleX, color: "text-red-700" },
  unknown: { label: "Unknown", Icon: CircleHelp, color: "text-gray-600" },
} as const;

interface StatusIndicatorProps {
  name: string;
  state: ServiceState;
}

/** One service and its state, as a labelled region. */
export function StatusIndicator({ name, state }: StatusIndicatorProps) {
  const headingId = useId();
  const { label, Icon, color } = presentation[state];

  return (
    <section
      aria-labelledby={headingId}
      className="flex items-center justify-between rounded-lg border border-gray-200 bg-white p-4"
    >
      <h2 id={headingId} className="font-medium">
        {name}
      </h2>
      <p className={clsx("flex items-center gap-2 text-sm font-medium", color)}>
        <Icon aria-hidden="true" className="size-5" />
        {label}
      </p>
    </section>
  );
}
