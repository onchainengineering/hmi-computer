import type { DeploymentConfig } from "api/api";
import { deploymentConfig } from "api/queries/deployment";
import type { AuthorizationResponse } from "api/typesGenerated";
import { ErrorAlert } from "components/Alert/ErrorAlert";
import { Loader } from "components/Loader/Loader";
import { Margins } from "components/Margins/Margins";
import { Stack } from "components/Stack/Stack";
import { useAuthenticated } from "contexts/auth/RequireAuth";
import { RequirePermission } from "contexts/auth/RequirePermission";
import type { Permissions } from "contexts/auth/permissions";
import { useDashboard } from "modules/dashboard/useDashboard";
import { type FC, Suspense, createContext, useContext } from "react";
import { useQuery } from "react-query";
import { Outlet, useLocation } from "react-router-dom";
import { Sidebar } from "./Sidebar";

type ManagementSettingsValue = Readonly<{
	deploymentValues: DeploymentConfig | undefined;
}>;

export const ManagementSettingsContext = createContext<
	ManagementSettingsValue | undefined
>(undefined);

export const useManagementSettings = (): ManagementSettingsValue => {
	const context = useContext(ManagementSettingsContext);
	if (!context) {
		throw new Error(
			"useManagementSettings should be used inside of ManagementSettingsLayout",
		);
	}

	return context;
};

/**
 * Return true if the user can edit the organization settings or its members.
 */
export const canEditOrganization = (
	permissions: AuthorizationResponse | undefined,
) => {
	return (
		permissions !== undefined &&
		(permissions.editOrganization ||
			permissions.editMembers ||
			permissions.editGroups)
	);
};

export const isManagementRoutePermitted = (
	locationPath: string,
	permissions: Permissions,
	showOrganizations: boolean,
): boolean => {
	if (!locationPath.startsWith("/")) {
		return false;
	}

	if (locationPath.startsWith("/organizations")) {
		return showOrganizations;
	}

	if (!locationPath.startsWith("/deployment")) {
		return false;
	}

	// Logic for deployment routes should mirror the conditions used to display
	// the sidebar tabs from SidebarView.tsx
	const href = locationPath.replace(/^\/deployment/, "");

	if (href === "/" || href === "") {
		const hasAtLeastOnePermission = Object.values(permissions).some((v) => v);
		return hasAtLeastOnePermission;
	}
	if (href.startsWith("/general")) {
		return permissions.viewDeploymentValues;
	}
	if (href.startsWith("/licenses")) {
		return permissions.viewAllLicenses;
	}
	if (href.startsWith("/appearance")) {
		return permissions.editDeploymentValues;
	}
	if (href.startsWith("/userauth")) {
		return permissions.viewDeploymentValues;
	}
	if (href.startsWith("/external-auth")) {
		return permissions.viewDeploymentValues;
	}
	if (href.startsWith("/network")) {
		return permissions.viewDeploymentValues;
	}
	if (href.startsWith("/workspace-proxies")) {
		return permissions.readWorkspaceProxies;
	}
	if (href.startsWith("/security")) {
		return permissions.viewDeploymentValues;
	}
	if (href.startsWith("/observability")) {
		return permissions.viewDeploymentValues;
	}
	if (href.startsWith("/users")) {
		return permissions.viewAllUsers;
	}
	if (href.startsWith("/notifications")) {
		return permissions.viewNotificationTemplate;
	}
	if (href.startsWith("/oauth2-provider")) {
		return permissions.viewExternalAuthConfig;
	}

	return false;
};

/**
 * A multi-org capable settings page layout.
 *
 * If multi-org is not enabled or licensed, this is the wrong layout to use.
 * See DeploySettingsLayoutInner instead.
 */
export const ManagementSettingsLayout: FC = () => {
	const location = useLocation();
	const { showOrganizations } = useDashboard();
	const { permissions } = useAuthenticated();
	const deploymentConfigQuery = useQuery({
		...deploymentConfig(),
		enabled: permissions.viewDeploymentValues,
	});

	// Have to make check more specific, because if the query is disabled, it
	// will be forever stuck in the loading state. The loading state is only
	// relevant to owners right now
	if (permissions.viewDeploymentValues && deploymentConfigQuery.isLoading) {
		return <Loader />;
	}

	const canViewAtLeastOneTab =
		permissions.viewDeploymentValues ||
		permissions.viewAllUsers ||
		permissions.editAnyOrganization ||
		permissions.viewAnyAuditLog;

	return (
		<RequirePermission
			permitted={
				canViewAtLeastOneTab &&
				isManagementRoutePermitted(
					location.pathname,
					permissions,
					showOrganizations,
				)
			}
			unpermittedRedirect={
				canViewAtLeastOneTab && !location.pathname.startsWith("/deployment")
					? "/deployment"
					: "/workspaces"
			}
		>
			<Margins>
				<Stack css={{ padding: "48px 0" }} direction="row" spacing={6}>
					<Sidebar />

					<main css={{ flexGrow: 1 }}>
						{deploymentConfigQuery.isError && (
							<ErrorAlert error={deploymentConfigQuery.error} />
						)}

						<ManagementSettingsContext.Provider
							value={{ deploymentValues: deploymentConfigQuery.data }}
						>
							<Suspense fallback={<Loader />}>
								<Outlet />
							</Suspense>
						</ManagementSettingsContext.Provider>
					</main>
				</Stack>
			</Margins>
		</RequirePermission>
	);
};
