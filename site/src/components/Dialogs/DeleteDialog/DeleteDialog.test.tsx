import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { render } from "testHelpers/renderHelpers";
import { DeleteDialog } from "./DeleteDialog";
import { act } from "react-dom/test-utils";

const inputTestId = "delete-dialog-name-confirmation";

async function fillInputField(inputElement: HTMLElement, text: string) {
  // 2023-10-06 - There's something wonky with MUI's ConfirmDialog that causes
  // its state to update after a typing event being fired, and React Testing
  // Library isn't able to catch it, making React DOM freak out because an
  // "unexpected" state change happened. Tried everything under the sun to catch
  // the state changes the proper way, but the only way to get around it for now
  // might be to manually make React aware of the changes

  // eslint-disable-next-line testing-library/no-unnecessary-act -- have to make sure state updates don't slip through cracks
  return act(() => userEvent.type(inputElement, text));
}

describe("DeleteDialog", () => {
  it("disables confirm button when the text field is empty", () => {
    render(
      <DeleteDialog
        isOpen
        onConfirm={jest.fn()}
        onCancel={jest.fn()}
        entity="template"
        name="MyTemplate"
      />,
    );

    const confirmButton = screen.getByRole("button", { name: "Delete" });
    expect(confirmButton).toBeDisabled();
  });

  it("disables confirm button when the text field is filled incorrectly", async () => {
    render(
      <DeleteDialog
        isOpen
        onConfirm={jest.fn()}
        onCancel={jest.fn()}
        entity="template"
        name="MyTemplate"
      />,
    );

    const textField = screen.getByTestId(inputTestId);
    await fillInputField(textField, "MyTemplateButWrong");

    const confirmButton = screen.getByRole("button", { name: "Delete" });
    expect(confirmButton).toBeDisabled();
  });

  it("enables confirm button when the text field is filled correctly", async () => {
    render(
      <DeleteDialog
        isOpen
        onConfirm={jest.fn()}
        onCancel={jest.fn()}
        entity="template"
        name="MyTemplate"
      />,
    );

    const textField = screen.getByTestId(inputTestId);
    await fillInputField(textField, "MyTemplate");

    const confirmButton = screen.getByRole("button", { name: "Delete" });
    expect(confirmButton).not.toBeDisabled();
  });
});
