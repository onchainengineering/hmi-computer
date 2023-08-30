import type {
  Reporter,
  FullConfig,
  Suite,
  TestCase,
  TestResult,
  FullResult,
} from "@playwright/test/reporter"

class CoderReporter implements Reporter {
  onBegin(config: FullConfig, suite: Suite) {
    // eslint-disable-next-line no-console -- Helpful for debugging
    console.log(`Starting the run with ${suite.allTests().length} tests`)
  }

  onTestBegin(test: TestCase) {
    // eslint-disable-next-line no-console -- Helpful for debugging
    console.log(`Starting test ${test.title}`)
  }

  onStdOut(chunk: string, test: TestCase, _: TestResult): void {
    // eslint-disable-next-line no-console -- Helpful for debugging
    console.log(
      `[stdout] [${test ? test.title : "unknown"}]: ${chunk.replace(
        /\n$/g,
        "",
      )}`,
    )
  }

  onStdErr(chunk: string, test: TestCase, _: TestResult): void {
    // eslint-disable-next-line no-console -- Helpful for debugging
    console.log(
      `[stderr] [${test ? test.title : "unknown"}]: ${chunk.replace(
        /\n$/g,
        "",
      )}`,
    )
  }

  onTestEnd(test: TestCase, result: TestResult) {
    // eslint-disable-next-line no-console -- Helpful for debugging
    console.log(`Finished test ${test.title}: ${result.status}`)
  }

  onEnd(result: FullResult) {
    // eslint-disable-next-line no-console -- Helpful for debugging
    console.log(`Finished the run: ${result.status}`)
  }
}
export default CoderReporter
