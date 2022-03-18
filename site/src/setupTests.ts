import { server } from "./test_helpers/server"

// Establish API mocking before all tests.
beforeAll(() =>
  server.listen({
    onUnhandledRequest: "warn",
  }),
)

// Reset any request handlers that we may add during the tests,
// so they don't affect other tests.
afterEach(() => server.resetHandlers())

// Clean up after the tests are finished.
afterAll(() => server.close())
