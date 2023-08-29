BEGIN;

ALTER TABLE templates RENAME COLUMN restart_requirement_days_of_week TO autostop_requirement_days_of_week;

ALTER TABLE templates RENAME COLUMN restart_requirement_weeks TO autostop_requirement_weeks;

DROP VIEW template_with_users;

CREATE VIEW
    template_with_users
AS
    SELECT
        templates.*,
		coalesce(visible_users.avatar_url, '') AS created_by_avatar_url,
		coalesce(visible_users.username, '') AS created_by_username
    FROM
        templates
    LEFT JOIN
		visible_users
	ON
	    templates.created_by = visible_users.id;

COMMENT ON VIEW template_with_users IS 'Joins in the username + avatar url of the created by user.';

COMMIT;
