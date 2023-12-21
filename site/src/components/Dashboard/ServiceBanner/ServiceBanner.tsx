import { type FC } from "react";
import { useDashboard } from "components/Dashboard/DashboardProvider";
import { ServiceBannerView } from "./ServiceBannerView";

export const ServiceBanner: FC = () => {
  const { appearance } = useDashboard();
  const { message, background_color, enabled } =
    appearance.config.service_banner;

  if (!enabled) {
    return null;
  }

  if (message === undefined || background_color === undefined) return null;

  return (
    <ServiceBannerView
      message={message}
      backgroundColor={background_color}
      isPreview={appearance.isPreview}
    />
  );
};
