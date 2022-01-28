import { firstOrItem } from "./array"

describe("array", () => {
  describe("firstOrItem", () => {
    it("returns null if empty array", () => {
      expect(firstOrItem([])).toBeNull()
    })

    it("returns first item if array with more one item", () => {
      expect(firstOrItem(["a", "b"])).toEqual("a")
    })

    it("returns item if single item", () => {
      expect(firstOrItem("c")).toEqual("c")
    })
  })
})
