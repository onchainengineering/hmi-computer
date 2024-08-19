import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { API } from "api/api";
import { TemplateLayout } from "pages/TemplatePage/TemplateLayout";
import {
	MockTemplate,
	MockTemplateVersionParameter1 as parameter1,
	MockTemplateVersionParameter2 as parameter2,
} from "testHelpers/entities";
import {
	renderWithAuth,
	waitForLoaderToBeRemoved,
} from "testHelpers/renderHelpers";
import TemplateEmbedPage from "./TemplateEmbedPage";

test("Users can fill the parameters and copy the open in coder url", async () => {
	jest
		.spyOn(API, "getTemplateVersionRichParameters")
		.mockResolvedValue([parameter1, parameter2]);

	renderWithAuth(
		<TemplateLayout>
			<TemplateEmbedPage />
		</TemplateLayout>,
		{
			route: `/templates/${MockTemplate.organization_name}/${MockTemplate.name}/embed`,
			path: "/templates/:organization/:template/embed",
		},
	);
	await waitForLoaderToBeRemoved();

	const user = userEvent.setup();
	const firstParameterField = screen.getByLabelText(
		parameter1.display_name ?? parameter1.name,
		{ exact: false },
	);
	await user.clear(firstParameterField);
	await user.type(firstParameterField, "firstParameterValue");
	const secondParameterField = screen.getByLabelText(
		parameter2.display_name ?? parameter2.name,
		{ exact: false },
	);
	await user.clear(secondParameterField);
	await user.type(secondParameterField, "123456");

	jest.spyOn(window.navigator.clipboard, "writeText");
	const copyButton = screen.getByRole("button", { name: /copy/i });
	await userEvent.click(copyButton);
	expect(window.navigator.clipboard.writeText).toBeCalledWith(
		`[![Open in Coder](http://localhost/open-in-coder.svg)](http://localhost/templates/${MockTemplate.organization_name}/${MockTemplate.name}/workspace?mode=manual&param.first_parameter=firstParameterValue&param.second_parameter=123456)`,
	);
});
