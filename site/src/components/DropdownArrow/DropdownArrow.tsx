import KeyboardArrowDown from "@mui/icons-material/KeyboardArrowDown";
import KeyboardArrowUp from "@mui/icons-material/KeyboardArrowUp";
import { type FC } from "react";
import { type Theme } from "@emotion/react";

interface ArrowProps {
  margin?: boolean;
  color?: string;
  close?: boolean;
}

export const DropdownArrow: FC<ArrowProps> = (props) => {
  const { margin = true, color, close } = props;

  const Arrow = close ? KeyboardArrowUp : KeyboardArrowDown;

  return (
    <Arrow
      aria-label={close ? "close-dropdown" : "open-dropdown"}
      css={(theme: Theme) => ({
        color: color ?? theme.deprecated.palette.primary.contrastText,
        marginLeft: margin ? 8 : 0,
        width: 16,
        height: 16,
      })}
    />
  );
};
