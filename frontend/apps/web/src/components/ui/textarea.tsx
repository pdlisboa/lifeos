import { forwardRef, type TextareaHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement>>(
  ({ className, ...props }, ref) => (
    <textarea
      ref={ref}
      className={cn(
        "w-full rounded-md border border-border bg-bg-raised px-3 py-2 text-sm text-fg-primary",
        "placeholder:text-fg-muted focus:outline-none focus:ring-2 focus:ring-delta-positive/50",
        className,
      )}
      {...props}
    />
  ),
);
Textarea.displayName = "Textarea";
