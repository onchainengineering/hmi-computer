import { Box, CircularProgress, makeStyles } from "@material-ui/core"
import React from "react"
import { RequestState } from "../../hooks/useRequest"

export interface LoadingPageProps<T> {
  request: RequestState<T>
  children: (state: T) => React.ReactElement<any, any>
}

const useStyles = makeStyles(() => ({
  fullScreenLoader: {
    position: "absolute",
    top: "0",
    left: "0",
    right: "0",
    bottom: "0",
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
  },
}))

export const LoadingPage: React.FC<LoadingPageProps<T>> = <T,>(props: LoadingPageProps<T>) => {
  const styles = useStyles()

  const { request, children } = props
  const { state } = request
  switch (state) {
    case "error":
      return <div>{request.error.toString()}</div>
    case "loading":
      return (
        <div className={styles.fullScreenLoader}>
          {" "}
          <CircularProgress />
        </div>
      )
    case "success":
      return children(request.payload)
  }
}
