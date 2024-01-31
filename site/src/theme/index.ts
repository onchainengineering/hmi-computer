import type { Theme as MuiTheme } from "@mui/material/styles";
import type * as monaco from "monaco-editor";
import type { NewTheme } from "./experimental";
import type { ExternalImageModeStyles } from "./externalImages";
import type { Roles } from "./roles";
import dark from "./dark";
import darkBlue from "./darkBlue";
import light from "./light";

export interface Theme extends MuiTheme {
  experimental: NewTheme;
  roles: Roles;
  monaco: monaco.editor.IStandaloneThemeData;
  externalImages: ExternalImageModeStyles;
}

interface RealThemeSeriously
  extends Pick<Theme, "monaco" | "externalImages">,
    NewTheme {}

const x = {} as unknown as RealThemeSeriously;

export const DEFAULT_THEME = "dark";

const theme = {
  dark,
  darkBlue,
  light,
} satisfies Record<string, Theme>;

export default theme;
