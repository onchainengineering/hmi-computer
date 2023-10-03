import { Meta, StoryObj } from "@storybook/react";
import { OverflowY } from "./OverflowY";

const numbers: number[] = [];
for (let i = 0; i < 20; i++) {
  numbers.push(i + 1);
}

const meta: Meta<typeof OverflowY> = {
  title: `components/${OverflowY.name}`,
  component: OverflowY,
  args: {
    maxHeight: 400,
    children: numbers.map((num, i) => (
      <p
        key={num}
        style={{
          height: "50px",
          padding: 0,
          margin: 0,
          backgroundColor: i % 2 === 0 ? "white" : "gray",
        }}
      >
        Element {num}
      </p>
    )),
  },
};

export default meta;

type Story = StoryObj<typeof OverflowY>;
export const Example: Story = {};
