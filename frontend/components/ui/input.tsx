import * as React from "react";
import { cn } from "@/lib/utils";

type InputProps = React.InputHTMLAttributes<HTMLInputElement> & {
  label: string;
};

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, id, ...props }, ref) => {
    const inputID = id ?? props.name;

    return (
      <label className="grid gap-2 text-sm font-medium text-foreground" htmlFor={inputID}>
        <span>{label}</span>
        <input
          ref={ref}
          id={inputID}
          className={cn(
            "h-11 w-full rounded-lg border border-border bg-surface px-3 text-sm text-foreground outline-none transition placeholder:text-muted focus:border-primary focus:ring-2 focus:ring-primary/20",
            className
          )}
          {...props}
        />
      </label>
    );
  }
);

Input.displayName = "Input";

