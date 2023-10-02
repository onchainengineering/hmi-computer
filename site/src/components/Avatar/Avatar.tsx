// This is the only place MuiAvatar can be used
// eslint-disable-next-line no-restricted-imports -- Read above
import MuiAvatar, { AvatarProps as MuiAvatarProps } from "@mui/material/Avatar";
import { FC } from "react";
import { css, type Theme } from "@emotion/react";

export type AvatarProps = MuiAvatarProps & {
  size?: "sm" | "md" | "xl";
  colorScheme?: "light" | "darken";
  fitImage?: boolean;
};

const sizeStyles = {
  sm: (theme: Theme) => ({
    width: theme.spacing(3),
    height: theme.spacing(3),
    fontSize: theme.spacing(1.5),
  }),
  md: {},
  xl: (theme: Theme) => ({
    width: theme.spacing(6),
    height: theme.spacing(6),
    fontSize: theme.spacing(3),
  }),
};

const colorStyles = {
  light: {},
  darken: (theme: Theme) => ({
    background: theme.palette.divider,
    color: theme.palette.text.primary,
  }),
};

const fitImageStyles = css`
  & .MuiAvatar-img {
    object-fit: contain;
  }
`;

export const Avatar: FC<AvatarProps> = ({
  size = "md",
  colorScheme = "light",
  fitImage,
  children,
  ...muiProps
}) => {
  return (
    <MuiAvatar
      {...muiProps}
      css={[
        sizeStyles[size],
        colorStyles[colorScheme],
        fitImage && fitImageStyles,
      ]}
    >
      {typeof children === "string" ? firstLetter(children) : children}
    </MuiAvatar>
  );
};

/**
 * Use it to make an img element behaves like a MaterialUI Icon component
 */
export const AvatarIcon: FC<{ src: string }> = ({ src }) => {
  return (
    <img
      src={src}
      alt=""
      css={{
        maxWidth: "50%",
      }}
    />
  );
};

const firstLetter = (str: string): string => {
  if (str.length > 0) {
    return str[0].toLocaleUpperCase();
  }

  return "";
};
