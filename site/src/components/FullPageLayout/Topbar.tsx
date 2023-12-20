import { css } from "@emotion/css";
import Button, { ButtonProps } from "@mui/material/Button";
import IconButton, { IconButtonProps } from "@mui/material/IconButton";
import { useTheme } from "@mui/material/styles";
import { Avatar, AvatarProps } from "components/Avatar/Avatar";
import {
  HTMLAttributes,
  PropsWithChildren,
  ReactElement,
  cloneElement,
  forwardRef,
} from "react";

export const Topbar = (props: HTMLAttributes<HTMLDivElement>) => {
  const theme = useTheme();

  return (
    <header
      {...props}
      css={{
        height: 48,
        borderBottom: `1px solid ${theme.palette.divider}`,
        display: "flex",
        alignItems: "center",
        fontSize: 13,
        lineHeight: "1.2",
      }}
    />
  );
};

export const TopbarIconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  (props, ref) => {
    return (
      <IconButton
        ref={ref}
        {...props}
        size="small"
        css={{
          padding: 0,
          borderRadius: 0,
          height: 48,
          width: 48,

          "& svg": {
            fontSize: 20,
          },
        }}
      />
    );
  },
) as typeof IconButton;

export const TopbarButton = forwardRef<HTMLButtonElement, ButtonProps>(
  (props: ButtonProps, ref) => {
    return (
      <Button
        ref={ref}
        color="neutral"
        css={{
          height: 28,
          fontSize: 13,
          borderRadius: 4,
          padding: "0 12px",
        }}
        {...props}
      />
    );
  },
);

export const TopbarData = (props: HTMLAttributes<HTMLDivElement>) => {
  return (
    <div
      {...props}
      css={{
        display: "flex",
        gap: 8,
        alignItems: "center",
        justifyContent: "center",
      }}
    />
  );
};

export const TopbarDivider = (props: HTMLAttributes<HTMLSpanElement>) => {
  const theme = useTheme();
  return (
    <span {...props} css={{ color: theme.palette.divider }}>
      /
    </span>
  );
};

export const TopbarAvatar = (props: AvatarProps) => {
  return (
    <Avatar
      {...props}
      variant="square"
      fitImage
      css={{ width: 16, height: 16 }}
    />
  );
};

export const TopbarIcon = ({
  children,
  ...props
}: PropsWithChildren<HTMLAttributes<HTMLOrSVGElement>>) => {
  const theme = useTheme();

  return cloneElement(
    children as ReactElement<HTMLAttributes<HTMLOrSVGElement>>,
    {
      ...props,
      className: css({ fontSize: 16, color: theme.palette.text.secondary }),
    },
  );
};
