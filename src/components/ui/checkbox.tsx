"use client";

import { Check } from "lucide-react";

import { cn } from "@/lib/utils";

interface CheckboxProps {
  checked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
  disabled?: boolean;
  className?: string;
}

function Checkbox({ checked = false, onCheckedChange, disabled, className }: CheckboxProps) {
  return (
    <button
      type="button"
      role="checkbox"
      aria-checked={checked}
      aria-disabled={disabled}
      disabled={disabled}
      onClick={() => !disabled && onCheckedChange?.(!checked)}
      className={cn(
        "inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-md border-2 outline-none transition-all focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer sm:h-4.5 sm:w-4.5",
        checked
          ? "border-primary bg-primary"
          : "border-input bg-background",
        className,
      )}
      data-slot="checkbox"
    >
      <Check
        className={cn(
          "h-3.5 w-3.5 stroke-[3] text-primary-foreground transition-opacity sm:h-3 sm:w-3",
          checked ? "opacity-100" : "opacity-0",
        )}
      />
    </button>
  );
}

export { Checkbox };

