import TextField from "@mui/material/TextField";
import { FormikContextType, useFormik } from "formik";
import { FC, useEffect, useState } from "react";
import * as Yup from "yup";
import { getFormHelpers } from "utils/formUtils";
import { LoadingButton } from "components/LoadingButton/LoadingButton";
import { ErrorAlert } from "components/Alert/ErrorAlert";
import { Form, FormFields } from "components/Form/Form";
import {
  UpdateUserQuietHoursScheduleRequest,
  UserQuietHoursScheduleResponse,
} from "api/typesGenerated";
import MenuItem from "@mui/material/MenuItem";
import { Stack } from "components/Stack/Stack";
import { timeZones, getPreferredTimezone } from "utils/timeZones";
import { Alert } from "components/Alert/Alert";
import { timeToCron, quietHoursDisplay } from "utils/schedule";

export interface ScheduleFormValues {
  startTime: string;
  timezone: string;
}

const validationSchema = Yup.object({
  startTime: Yup.string()
    .ensure()
    .test("is-time-string", "Time must be in HH:mm format.", (value) => {
      if (value === "") {
        return true;
      }
      if (!/^[0-9][0-9]:[0-9][0-9]$/.test(value)) {
        return false;
      }
      const parts = value.split(":");
      const HH = Number(parts[0]);
      const mm = Number(parts[1]);
      return HH >= 0 && HH <= 23 && mm >= 0 && mm <= 59;
    }),
  timezone: Yup.string().required(),
});

export interface ScheduleFormProps {
  isLoading: boolean;
  initialValues: UserQuietHoursScheduleResponse;
  mutationError: unknown;
  onSubmit: (data: UpdateUserQuietHoursScheduleRequest) => void;
  // now can be set to force the time used for "Next occurrence" in tests.
  now?: Date;
}

export const ScheduleForm: FC<React.PropsWithChildren<ScheduleFormProps>> = ({
  isLoading,
  initialValues,
  mutationError,
  onSubmit,
  now,
}) => {
  // Update every 15 seconds to update the "Next occurrence" field.
  const [, setTime] = useState<number>(Date.now());
  useEffect(() => {
    const interval = setInterval(() => setTime(Date.now()), 15000);
    return () => {
      clearInterval(interval);
    };
  }, []);

  const preferredTimezone = getPreferredTimezone();

  // If the user has a custom schedule, use that as the initial values.
  // Otherwise, use midnight in their preferred timezone.
  const formInitialValues = {
    startTime: "00:00",
    timezone: preferredTimezone,
  };
  if (initialValues.user_set) {
    formInitialValues.startTime = initialValues.time;
    formInitialValues.timezone = initialValues.timezone;
  }

  const form: FormikContextType<ScheduleFormValues> =
    useFormik<ScheduleFormValues>({
      initialValues: formInitialValues,
      validationSchema,
      onSubmit: async (values) => {
        onSubmit({
          schedule: timeToCron(values.startTime, values.timezone),
        });
      },
    });
  const getFieldHelpers = getFormHelpers<ScheduleFormValues>(
    form,
    mutationError,
  );

  return (
    <Form onSubmit={form.handleSubmit}>
      <FormFields>
        {Boolean(mutationError) && <ErrorAlert error={mutationError} />}

        {!initialValues.user_set && (
          <Alert severity="info">
            You are currently using the default quiet hours schedule, which
            starts every day at <code>{initialValues.time}</code> in{" "}
            <code>{initialValues.timezone}</code>.
          </Alert>
        )}

        <Stack direction="row">
          <TextField
            {...getFieldHelpers("startTime")}
            disabled={isLoading}
            label="Start time"
            type="time"
            fullWidth
          />
          <TextField
            {...getFieldHelpers("timezone")}
            disabled={isLoading}
            label="Timezone"
            select
            fullWidth
          >
            {timeZones.map((zone) => (
              <MenuItem key={zone} value={zone}>
                {zone}
              </MenuItem>
            ))}
          </TextField>
        </Stack>

        <TextField
          disabled
          fullWidth
          label="Cron schedule"
          value={timeToCron(form.values.startTime, form.values.timezone)}
        />

        <TextField
          disabled
          fullWidth
          label="Next occurrence"
          value={quietHoursDisplay(
            form.values.startTime,
            form.values.timezone,
            now,
          )}
        />

        <div>
          <LoadingButton
            loading={isLoading}
            disabled={isLoading}
            type="submit"
            variant="contained"
          >
            {!isLoading && "Update schedule"}
          </LoadingButton>
        </div>
      </FormFields>
    </Form>
  );
};
