import TableCell from "@mui/material/TableCell";
import TableRow from "@mui/material/TableRow";
import { type FC } from "react";
import { css, useTheme } from "@emotion/react";
import { createDisplayDate } from "./utils";

export interface TimelineDateRow {
  date: Date;
}

export const TimelineDateRow: FC<TimelineDateRow> = ({ date }) => {
  const theme = useTheme();

  return (
    <TableRow
      css={css`
        background: ${theme.palette.background.paper};

        &:not(:first-of-type) td {
          border-top: 1px solid ${theme.palette.divider};
        }
      `}
    >
      <TableCell
        css={{
          padding: `${theme.spacing(1, 4)} !important`,
          background: `${theme.palette.background.paperLight} !important`,
          fontSize: 12,
          position: "relative",
          color: theme.palette.text.secondary,
          textTransform: "capitalize",
        }}
        title={date.toLocaleDateString()}
      >
        {createDisplayDate(date)}
      </TableCell>
    </TableRow>
  );
};
