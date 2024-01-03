/**
 * @file A more sophisticated version of the native <abbr> element.
 *
 * Features:
 * - Better type-safety (requiring you to include certain properties)
 * - All built-in HTML styling is stripped away by default
 * - Better integration with screen readers (making the title prop available),
 *   with more options for controlling how they read out initialisms
 */
import { visuallyHidden } from "@mui/utils";
import { type FC, type HTMLAttributes } from "react";

type AbbrProps = HTMLAttributes<HTMLElement> & {
  children: string;
  title: string;
  pronunciation?: "shorthand" | "acronym" | "initialism";
};

export const Abbr: FC<AbbrProps> = ({
  children,
  title,
  pronunciation = "shorthand",
  ...delegatedProps
}) => {
  return (
    <abbr
      // Title attributes usually aren't natively available to screen readers;
      // still have to inject text manually. Main value of titles here is
      // letting sighted users hover over the abbreviation to see the full term
      title={title}
      data-testid="abbr"
      css={{
        textDecoration: "inherit",
        letterSpacing: isAllUppercase(children) ? "0.02em" : "0",
      }}
      {...delegatedProps}
    >
      {pronunciation === "shorthand" ? (
        <>{children}</>
      ) : (
        // Helps make sure that screen readers read initialisms/acronyms
        // correctly without it affecting the visual output for sighted users
        // (e.g., Mac VoiceOver reads "CLI" as "klee" by default)
        <>
          {/*Can be simplified once CSS "spell-out" has more browser support*/}
          <span css={{ ...visuallyHidden }} data-testid="visually-hidden">
            {pronunciation === "initialism"
              ? initializeText(children)
              : flattenPronunciation(children)}
          </span>

          <span aria-hidden data-testid="visual-only">
            {children}
          </span>
        </>
      )}
    </abbr>
  );
};

function initializeText(text: string): string {
  return text.trim().toUpperCase().replaceAll(/\B/g, ".") + ".";
}

function flattenPronunciation(text: string): string {
  const trimmed = text.trim();
  return (trimmed[0] ?? "").toUpperCase() + trimmed.slice(1).toLowerCase();
}

function isAllUppercase(text: string): boolean {
  return text === text.toUpperCase();
}
