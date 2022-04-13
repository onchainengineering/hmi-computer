/**
 * @fileoverview This file is configures Storybook
 *
 * @see <https://storybook.js.org/docs/react/configure/overview>
 */
const path = require("path")

module.exports = {
  // Automatically loads all stories in source ending in 'stories.tsx'
  //
  // SEE: https://storybook.js.org/docs/react/configure/overview#configure-story-loading
  stories: ["../src/**/*.stories.tsx"],

  // addons are official and community plugins to extend Storybook.
  //
  // SEE: https://storybook.js.org/addons
  addons: ["@storybook/addon-links", "@storybook/addon-essentials", "@react-theming/storybook-addon"],

  // Storybook uses babel under the hood, while we currently use ts-loader.
  // Sometimes, you may encounter an error in a Storybook that contains syntax
  // that requires a babel plugin.
  //
  // SEE: https://storybook.js.org/docs/react/configure/babel
  babel: async (options) => ({
    ...options,
    plugins: ["@babel/plugin-proposal-class-properties"],
  }),

  // Storybook internally uses its own Webpack configuration instead of ours.
  //
  // SEE: https://storybook.js.org/docs/react/configure/webpack
  webpackFinal: async (config) => {
    config.resolve.modules = [path.resolve(__dirname, ".."), "node_modules"]
    return config
  },
}
