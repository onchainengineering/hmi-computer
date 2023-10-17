import { useTheme } from "@emotion/react";
import { type Group } from "api/typesGenerated";

import { Stack } from "components/Stack/Stack";
import TableCell from "@mui/material/TableCell";
import Button from "@mui/material/Button";
import { useState } from "react";

type GroupsCellProps = {
  userGroups: readonly Group[] | undefined;
};

export function GroupsCell({ userGroups }: GroupsCellProps) {
  const [isHovering, setIsHovering] = useState(false);
  const theme = useTheme();

  return (
    <TableCell>
      {userGroups === undefined ? (
        <em>N/A</em>
      ) : (
        <Button
          onPointerEnter={() => setIsHovering(true)}
          onPointerLeave={() => setIsHovering(false)}
          css={{
            border: "none",
            textAlign: "left",
            padding: 0,
            lineHeight: "1.4",
            "&:hover": {
              border: "none",
              backgroundColor: "transparent",
            },
          }}
        >
          <Stack spacing={0}>
            <span>
              {userGroups.length} Group{userGroups.length !== 1 && "s"}
            </span>

            <span
              css={{
                fontSize: "0.75rem",
                color: theme.palette.text.secondary,
                textDecoration: isHovering ? "none" : "underline",
                textUnderlineOffset: "0.2em",
              }}
            >
              See details
            </span>
          </Stack>
        </Button>
      )}
    </TableCell>
  );
}
