ALTER TABLE template_version_parameters DROP CONSTRAINT validation_monotonic_order;

ALTER TABLE template_version_parameters DROP COLUMN validation_monotonic;
