import {
  TemplateVersionParameter,
  WorkspaceBuildParameter,
} from "api/typesGenerated";
import * as Yup from "yup";

export type AutofillSource = "user_history" | "url" | "active_build";

// AutofillBuildParameter is a build parameter destined to a form, alongside
// its source so that the form can explain where the value comes from.
export type AutofillBuildParameter = {
  source: AutofillSource;
} & WorkspaceBuildParameter;

export const getInitialRichParameterValues = (
  templateParams: TemplateVersionParameter[],
  autofillParams?: AutofillBuildParameter[],
): WorkspaceBuildParameter[] => {
  return templateParams.map((parameter) => {
    // Short-circuit for ephemeral parameters, which are always reset to
    // the template-defined default.
    if (parameter.ephemeral) {
      return {
        name: parameter.name,
        value: parameter.default_value,
      };
    }

    const autofillParam = autofillParams?.find(
      (p) => p.name === parameter.name,
    );

    if (autofillParam !== undefined && isValidValue(parameter, autofillParam)) {
      return {
        name: parameter.name,
        value: autofillParam.value,
      };
    }

    return {
      name: parameter.name,
      value: parameter.default_value,
    };
  });
};

const isValidValue = (
  templateParam: TemplateVersionParameter,
  buildParam: WorkspaceBuildParameter,
) => {
  if (templateParam.options.length > 0) {
    const validValues = templateParam.options.map((option) => option.value);
    return validValues.includes(buildParam.value);
  }

  return true;
};

export const useValidationSchemaForRichParameters = (
  templateParameters?: TemplateVersionParameter[],
  lastBuildParameters?: WorkspaceBuildParameter[],
): Yup.AnySchema => {
  if (!templateParameters) {
    return Yup.object();
  }

  return Yup.array()
    .of(
      Yup.object().shape({
        name: Yup.string().required(),
        value: Yup.string().test("verify with template", (val, ctx) => {
          const name = ctx.parent.name;
          const templateParameter = templateParameters.find(
            (parameter) => parameter.name === name,
          );
          if (templateParameter) {
            switch (templateParameter.type) {
              case "number":
                if (
                  templateParameter.validation_min &&
                  !templateParameter.validation_max
                ) {
                  if (Number(val) < templateParameter.validation_min) {
                    return ctx.createError({
                      path: ctx.path,
                      message: `Value must be greater than ${templateParameter.validation_min}.`,
                    });
                  }
                } else if (
                  !templateParameter.validation_min &&
                  templateParameter.validation_max
                ) {
                  if (templateParameter.validation_max < Number(val)) {
                    return ctx.createError({
                      path: ctx.path,
                      message: `Value must be less than ${templateParameter.validation_max}.`,
                    });
                  }
                } else if (
                  templateParameter.validation_min &&
                  templateParameter.validation_max
                ) {
                  if (
                    Number(val) < templateParameter.validation_min ||
                    templateParameter.validation_max < Number(val)
                  ) {
                    return ctx.createError({
                      path: ctx.path,
                      message: `Value must be between ${templateParameter.validation_min} and ${templateParameter.validation_max}.`,
                    });
                  }
                }

                if (
                  templateParameter.validation_monotonic &&
                  lastBuildParameters
                ) {
                  const lastBuildParameter = lastBuildParameters.find(
                    (last) => last.name === name,
                  );
                  if (lastBuildParameter) {
                    switch (templateParameter.validation_monotonic) {
                      case "increasing":
                        if (Number(lastBuildParameter.value) > Number(val)) {
                          return ctx.createError({
                            path: ctx.path,
                            message: `Value must only ever increase (last value was ${lastBuildParameter.value})`,
                          });
                        }
                        break;
                      case "decreasing":
                        if (Number(lastBuildParameter.value) < Number(val)) {
                          return ctx.createError({
                            path: ctx.path,
                            message: `Value must only ever decrease (last value was ${lastBuildParameter.value})`,
                          });
                        }
                        break;
                    }
                  }
                }
                break;
              case "string":
                {
                  if (
                    !templateParameter.validation_regex ||
                    templateParameter.validation_regex.length === 0
                  ) {
                    return true;
                  }

                  const regex = new RegExp(templateParameter.validation_regex);
                  if (val && !regex.test(val)) {
                    return ctx.createError({
                      path: ctx.path,
                      message: `${templateParameter.validation_error} (value does not match the pattern ${templateParameter.validation_regex})`,
                    });
                  }
                }
                break;
            }
          }
          return true;
        }),
      }),
    )
    .required();
};
