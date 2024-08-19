import { forDarkThemes } from "../externalImages";
import experimental from "./experimental";
import monaco from "./monaco";
import muiTheme from "./mui";
import permission from "./permission";
import roles from "./roles";

export default {
	...muiTheme,
	externalImages: forDarkThemes,
	experimental,
	monaco,
	roles,
	permission,
};
