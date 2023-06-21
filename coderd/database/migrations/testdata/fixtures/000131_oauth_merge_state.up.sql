INSERT INTO oauth_merge_state(
	state_string,
	created_at,
	expires_at,
	to_login_type,
	user_id
) VALUES (
	gen_random_uuid()::text,
	now(),
	now() + interval '24 hour',
    'oidc',
	(SELECT id FROM users LIMIT 1)
)
