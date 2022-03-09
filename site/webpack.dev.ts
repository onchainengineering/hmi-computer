/**
 * @fileoverview This file contains a development configuration for webpack
 * meant for webpack-dev-server.
 */
import ReactRefreshWebpackPlugin from "@pmmmwh/react-refresh-webpack-plugin"
import { Configuration } from "webpack"
import "webpack-dev-server"

import { commonWebpackConfig } from "./webpack.common"

const commonPlugins = commonWebpackConfig.plugins || []

const config: Configuration = {
  ...commonWebpackConfig,

  // devtool controls how source maps are generated. In development, we want
  // more details (less optimized) for more readability and an easier time
  // debugging
  devtool: "eval-source-map",

  // devServer is the configuration for webpack-dev-server.
  //
  // REMARK: needs webpack-dev-server import at top of file for typings
  devServer: {
    allowedHosts: "all",

    // client configures options that are observed in the browser/web-client.
    client: {
      // automatically sets the browser console to verbose when using HMR
      logging: "verbose",

      // errors will display as a full-screen overlay
      overlay: true,

      // build % progress will display in the browser
      progress: true,

      // webpack-dev-server uses a webSocket to communicate with the browser
      // for HMR. By setting this to auto://0.0.0.0/ws, we allow the browser
      // to set the protocal, hostname and port automatically for us.
      webSocketURL: "auto://0.0.0.0:0/ws",
    },
    devMiddleware: {
      publicPath: "/",
    },
    headers: {
      "Access-Control-Allow-Origin": "*",
    },

    // historyApiFallback is required when using history (react-router) for
    // properly serving index.html on 404s.
    historyApiFallback: true,
    hot: true,
    port: 8080,
    proxy: {
      "/api": "http://localhost:3000",
    },
    static: ["./static"],
  },

  // Development mode - see:
  // https://webpack.js.org/configuration/mode/#mode-development
  mode: "development",
  plugins: [
    ...commonPlugins,

    // The ReactRefreshWebpackPlugin enables hot-module reloading:
    // https://github.com/pmmmwh/react-refresh-webpack-plugin
    new ReactRefreshWebpackPlugin({
      overlay: true,
    }),
  ],
}

export default config
