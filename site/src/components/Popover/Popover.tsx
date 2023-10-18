import {
  ReactElement,
  ReactNode,
  cloneElement,
  createContext,
  useContext,
  useRef,
  useState,
} from "react";
import MuiPopover, {
  type PopoverProps as MuiPopoverProps,
} from "@mui/material/Popover";

type TriggerMode = "hover" | "click";

type TriggerRef = React.RefObject<HTMLElement>;

type TriggerElement = ReactElement<{
  onClick?: () => void;
  ref: TriggerRef;
}>;

type PopoverContextValue = {
  open: boolean;
  setOpen: React.Dispatch<React.SetStateAction<boolean>>;
  triggerRef: TriggerRef;
  mode: TriggerMode;
};

const PopoverContext = createContext<PopoverContextValue | undefined>(
  undefined,
);

export const Popover = (props: {
  children: ReactNode | ((popover: PopoverContextValue) => ReactNode); // Allows inline usage
  mode?: TriggerMode;
  defaultOpen?: boolean;
}) => {
  const [open, setOpen] = useState(props.defaultOpen ?? false);
  const triggerRef = useRef<HTMLElement>(null);
  const value = { open, setOpen, triggerRef, mode: props.mode ?? "click" };

  return (
    <PopoverContext.Provider value={value}>
      {typeof props.children === "function"
        ? props.children(value)
        : props.children}
    </PopoverContext.Provider>
  );
};

export const usePopover = () => {
  const context = useContext(PopoverContext);
  if (!context) {
    throw new Error(
      "Popover compound components cannot be rendered outside the Popover component",
    );
  }
  return context;
};

export const PopoverTrigger = (props: {
  children: TriggerElement;
  hover?: boolean;
}) => {
  const popover = usePopover();

  const clickProps = {
    onClick: () => {
      popover.setOpen((open) => !open);
    },
  };

  const hoverProps = {
    onMouseEnter: () => {
      popover.setOpen(true);
    },
    onMouseLeave: () => {
      popover.setOpen(false);
    },
  };

  return cloneElement(props.children, {
    ...(popover.mode === "click" ? clickProps : hoverProps),
    ref: popover.triggerRef,
  });
};

type Horizontal = "left" | "right";

export const PopoverContent = (
  props: Omit<MuiPopoverProps, "open" | "onClose" | "anchorEl"> & {
    horizontal?: Horizontal;
  },
) => {
  const popover = usePopover();
  const horizontal = props.horizontal ?? "left";
  const hoverMode = popover.mode === "hover";

  return (
    <MuiPopover
      disablePortal
      css={(theme) => ({
        marginTop: hoverMode ? undefined : theme.spacing(1),
        pointerEvents: hoverMode ? "none" : undefined,
        "& .MuiPaper-root": {
          minWidth: theme.spacing(40),
          fontSize: 14,
          pointerEvents: hoverMode ? "auto" : undefined,
        },
      })}
      {...horizontalProps(horizontal)}
      {...modeProps(popover)}
      {...props}
      open={popover.open}
      onClose={() => popover.setOpen(false)}
      anchorEl={popover.triggerRef.current}
    />
  );
};

const modeProps = (popover: PopoverContextValue) => {
  if (popover.mode === "hover") {
    return {
      onMouseEnter: () => {
        popover.setOpen(true);
      },
      onMouseLeave: () => {
        popover.setOpen(false);
      },
    };
  }

  return {};
};

const horizontalProps = (horizontal: Horizontal) => {
  if (horizontal === "right") {
    return {
      anchorOrigin: {
        vertical: "bottom",
        horizontal: "right",
      },
      transformOrigin: {
        vertical: "top",
        horizontal: "right",
      },
    } as const;
  }

  return {
    anchorOrigin: {
      vertical: "bottom",
      horizontal: "left",
    },
  } as const;
};
