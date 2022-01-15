import express from "express"
import { createProxyMiddleware } from "http-proxy-middleware"
import next from "next"

const port = process.env.PORT || 8080
const dev = process.env.NODE_ENV !== "production"

if (!process.env.CODERV2_HOST) {
  throw new Error("CODERV2_HOST must be set")
} else if (!/^http(s)?:\/\//.test(process.env.CODERV2_HOST)) {
  throw new Error("CODERV2_HOST must be http(s)")
}

const app = next({ dev, dir: "./site" })
const handle = app.getRequestHandler()

app
  .prepare()
  .then(() => {
    const server = express()
    const paths: { [key: string]: Record<string, unknown> } = {
      "/proxy": {
        target: process.env.CODERV2_HOST,
        ws: true,
        secure: false,
        changeOrigin: true,
      },
      "/api": {
        target: process.env.CODERV2_HOST,
        ws: true,
        secure: false,
        changeOrigin: true,
      },
      "/auth": {
        target: process.env.CODERV2_HOST,
        ws: false,
        secure: false,
        changeOrigin: true,
      },
    }
    Object.keys(paths).forEach((k) => {
      server.use(k, createProxyMiddleware(k, paths[k]))
    })
    server.all("*", (req, res) => handle(req, res))
    server.listen(port)
  })
  .catch((err) => {
    console.error(err)
    process.exit(1)
  })
