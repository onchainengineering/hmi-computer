import * as SwitchPrimitives from "@radix-ui/react-switch";
import * as React from "react";

import { cn } from "utils/cn";

export const Switch = React.forwardRef<
	React.ElementRef<typeof SwitchPrimitives.Root>,
	React.ComponentPropsWithoutRef<typeof SwitchPrimitives.Root>
>(({ className, ...props }, ref) => (
	<SwitchPrimitives.Root
		className={cn(
			"peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-content-link focus-visible:ring-offset-2 focus-visible:ring-offset-surface-primary disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-surface-invert-primary data-[state=unchecked]:bg-surface-quaternary",
			className,
		)}
		{...props}
		ref={ref}
	>
		<SwitchPrimitives.Thumb
			className={cn(
				"pointer-events-none block h-4 w-4 rounded-full bg-surface-primary shadow-lg ring-0 transition-transform data-[state=checked]:translate-x-2.5 data-[state=unchecked]:-translate-x-1.5",
			)}
		/>
	</SwitchPrimitives.Root>
));
