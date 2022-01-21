import React from "react"
import CssBaseline from "@material-ui/core/CssBaseline"
import { makeStyles } from "@material-ui/core/styles"
import ThemeProvider from "@material-ui/styles/ThemeProvider"

import { light } from "../theme"
import { AppProps } from "next/app"
import { Navbar } from "../components/Navbar"
import { Footer } from "../components/Page"

/**
 * `Contents` is the wrapper around the core app UI,
 * containing common UI elements like the footer and navbar.
 *
 * This can't be inlined in `MyApp` because it requires styling,
 * and `useStyles` needs to be inside a `<ThemeProvider />`
 */
const Contents: React.FC<AppProps> = ({ Component, pageProps }) => {
  const styles = useStyles()

  const header = (
    <div className={styles.header}>
      <Navbar />
    </div>
  )

  const footer = (
    <div className={styles.footer}>
      <Footer />
    </div>
  )

  return (
    <div className={styles.root}>
      {header}
      <Component {...pageProps} />
      {footer}
    </div>
  )
}

/**
 * ClientRender is a component that only allows its children to be rendered
 * client-side. This check is performed by querying the existence of the window
 * global.
 */
const ClientRender: React.FC = ({ children }) => (
  <div suppressHydrationWarning>{typeof window === "undefined" ? null : children}</div>
)

/**
 * <App /> is the root rendering logic of the application - setting up our router
 * and any contexts / global state management.
 */
const MyApp: React.FC<AppProps> = ({ Component, pageProps }) => {
  return (
    <ClientRender>
      <ThemeProvider theme={light}>
        <CssBaseline />
        <Component {...pageProps} />
      </ThemeProvider>
    </ClientRender>
  )
}

const useStyles = makeStyles(() => ({
  root: {
    display: "flex",
    flexDirection: "column",
  },
  header: {
    flex: 0,
  },
  body: {
    height: "100%",
    flex: 1,
  },
  footer: {
    flex: 0,
  },
}))

export default MyApp
