// Package apidoc GENERATED BY SWAG; DO NOT EDIT
// This file was generated by swaggo/swag
package apidoc

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "termsOfService": "https://coder.com/legal/terms-of-service",
        "contact": {
            "name": "API Support",
            "url": "http://coder.com",
            "email": "support@coder.com"
        },
        "license": {
            "name": "AGPL-3.0",
            "url": "https://github.com/coder/coder/blob/main/LICENSE"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/organizations/{organization-id}/templates/": {
            "post": {
                "security": [
                    {
                        "CoderSessionToken": []
                    }
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Templates"
                ],
                "summary": "Create template by organization",
                "operationId": "create-template-by-organization",
                "parameters": [
                    {
                        "description": "Request body",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/codersdk.CreateTemplateRequest"
                        }
                    },
                    {
                        "type": "string",
                        "description": "Organization ID",
                        "name": "organization-id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/codersdk.Template"
                        }
                    }
                }
            }
        },
        "/templates/{id}": {
            "get": {
                "security": [
                    {
                        "CoderSessionToken": []
                    }
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Templates"
                ],
                "summary": "Get template metadata by ID",
                "operationId": "get-template-metadata-by-id",
                "parameters": [
                    {
                        "type": "string",
                        "format": "uuid",
                        "description": "Template ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/codersdk.Template"
                        }
                    }
                }
            }
        },
        "/workspaces": {
            "get": {
                "security": [
                    {
                        "CoderSessionToken": []
                    }
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Workspaces"
                ],
                "summary": "List workspaces",
                "operationId": "get-workspaces",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Filter by owner username",
                        "name": "owner",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Filter by template name",
                        "name": "template",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Filter with partial-match by workspace name",
                        "name": "name",
                        "in": "query"
                    },
                    {
                        "enum": [
                            "pending",
                            "running",
                            "stopping",
                            "stopped",
                            "failed",
                            "canceling",
                            "canceled",
                            "deleted",
                            "deleting"
                        ],
                        "type": "string",
                        "description": "Filter by workspace status",
                        "name": "status",
                        "in": "query"
                    },
                    {
                        "enum": [
                            "connected",
                            "connecting",
                            "disconnected",
                            "timeout"
                        ],
                        "type": "string",
                        "description": "Filter by agent status",
                        "name": "has_agent",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/codersdk.WorkspacesResponse"
                        }
                    }
                }
            }
        },
        "/workspaces/{id}": {
            "get": {
                "security": [
                    {
                        "CoderSessionToken": []
                    }
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Workspaces"
                ],
                "summary": "Get workspace metadata by ID",
                "operationId": "get-workspace-metadata-by-id",
                "parameters": [
                    {
                        "type": "string",
                        "format": "uuid",
                        "description": "Workspace ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Return data instead of HTTP 404 if the workspace is deleted",
                        "name": "include_deleted",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/codersdk.Workspace"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "codersdk.CreateParameterRequest": {
            "type": "object",
            "required": [
                "destination_scheme",
                "name",
                "source_scheme",
                "source_value"
            ],
            "properties": {
                "copy_from_parameter": {
                    "description": "CloneID allows copying the value of another parameter.\nThe other param must be related to the same template_id for this to\nsucceed.\nNo other fields are required if using this, as all fields will be copied\nfrom the other parameter.",
                    "type": "string"
                },
                "destination_scheme": {
                    "type": "string",
                    "enum": [
                        "environment_variable",
                        "provisioner_variable"
                    ]
                },
                "name": {
                    "type": "string"
                },
                "source_scheme": {
                    "type": "string",
                    "enum": [
                        "data"
                    ]
                },
                "source_value": {
                    "type": "string"
                }
            }
        },
        "codersdk.CreateTemplateRequest": {
            "type": "object",
            "required": [
                "name",
                "template_version_id"
            ],
            "properties": {
                "allow_user_cancel_workspace_jobs": {
                    "description": "Allow users to cancel in-progress workspace jobs.\n*bool as the default value is \"true\".",
                    "type": "boolean"
                },
                "default_ttl_ms": {
                    "description": "DefaultTTLMillis allows optionally specifying the default TTL\nfor all workspaces created from this template.",
                    "type": "integer"
                },
                "description": {
                    "description": "Description is a description of what the template contains. It must be\nless than 128 bytes.",
                    "type": "string"
                },
                "display_name": {
                    "description": "DisplayName is the displayed name of the template.",
                    "type": "string"
                },
                "icon": {
                    "description": "Icon is a relative path or external URL that specifies\nan icon to be displayed in the dashboard.",
                    "type": "string"
                },
                "name": {
                    "description": "Name is the name of the template.",
                    "type": "string"
                },
                "parameter_values": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/codersdk.CreateParameterRequest"
                    }
                },
                "template_version_id": {
                    "description": "VersionID is an in-progress or completed job to use as an initial version\nof the template.\n\nThis is required on creation to enable a user-flow of validating a\ntemplate works. There is no reason the data-model cannot support empty\ntemplates, but it doesn't make sense for users.",
                    "type": "string"
                }
            }
        },
        "codersdk.DERPRegion": {
            "type": "object",
            "properties": {
                "latency_ms": {
                    "type": "number"
                },
                "preferred": {
                    "type": "boolean"
                }
            }
        },
        "codersdk.Healthcheck": {
            "type": "object",
            "properties": {
                "interval": {
                    "description": "Interval specifies the seconds between each health check.",
                    "type": "integer"
                },
                "threshold": {
                    "description": "Threshold specifies the number of consecutive failed health checks before returning \"unhealthy\".",
                    "type": "integer"
                },
                "url": {
                    "description": "URL specifies the url to check for the app health.",
                    "type": "string"
                }
            }
        },
        "codersdk.NullTime": {
            "type": "object",
            "properties": {
                "time": {
                    "type": "string"
                },
                "valid": {
                    "description": "Valid is true if Time is not NULL",
                    "type": "boolean"
                }
            }
        },
        "codersdk.ProvisionerJob": {
            "type": "object",
            "properties": {
                "canceled_at": {
                    "type": "string"
                },
                "completed_at": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "error": {
                    "type": "string"
                },
                "file_id": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "started_at": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                },
                "tags": {
                    "type": "object",
                    "additionalProperties": {
                        "type": "string"
                    }
                },
                "worker_id": {
                    "type": "string"
                }
            }
        },
        "codersdk.Template": {
            "type": "object",
            "properties": {
                "active_user_count": {
                    "description": "ActiveUserCount is set to -1 when loading.",
                    "type": "integer"
                },
                "active_version_id": {
                    "type": "string"
                },
                "allow_user_cancel_workspace_jobs": {
                    "type": "boolean"
                },
                "build_time_stats": {
                    "$ref": "#/definitions/codersdk.TemplateBuildTimeStats"
                },
                "created_at": {
                    "type": "string"
                },
                "created_by_id": {
                    "type": "string"
                },
                "created_by_name": {
                    "type": "string"
                },
                "default_ttl_ms": {
                    "type": "integer"
                },
                "description": {
                    "type": "string"
                },
                "display_name": {
                    "type": "string"
                },
                "icon": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "name": {
                    "type": "string"
                },
                "organization_id": {
                    "type": "string"
                },
                "provisioner": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                },
                "workspace_owner_count": {
                    "type": "integer"
                }
            }
        },
        "codersdk.TemplateBuildTimeStats": {
            "type": "object",
            "additionalProperties": {
                "$ref": "#/definitions/codersdk.TransitionStats"
            }
        },
        "codersdk.TransitionStats": {
            "type": "object",
            "properties": {
                "p50": {
                    "type": "integer"
                },
                "p95": {
                    "type": "integer"
                }
            }
        },
        "codersdk.Workspace": {
            "type": "object",
            "properties": {
                "autostart_schedule": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "last_used_at": {
                    "type": "string"
                },
                "latest_build": {
                    "$ref": "#/definitions/codersdk.WorkspaceBuild"
                },
                "name": {
                    "type": "string"
                },
                "outdated": {
                    "type": "boolean"
                },
                "owner_id": {
                    "type": "string"
                },
                "owner_name": {
                    "type": "string"
                },
                "template_allow_user_cancel_workspace_jobs": {
                    "type": "boolean"
                },
                "template_display_name": {
                    "type": "string"
                },
                "template_icon": {
                    "type": "string"
                },
                "template_id": {
                    "type": "string"
                },
                "template_name": {
                    "type": "string"
                },
                "ttl_ms": {
                    "type": "integer"
                },
                "updated_at": {
                    "type": "string"
                }
            }
        },
        "codersdk.WorkspaceAgent": {
            "type": "object",
            "properties": {
                "apps": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/codersdk.WorkspaceApp"
                    }
                },
                "architecture": {
                    "type": "string"
                },
                "connection_timeout_seconds": {
                    "type": "integer"
                },
                "created_at": {
                    "type": "string"
                },
                "directory": {
                    "type": "string"
                },
                "disconnected_at": {
                    "type": "string"
                },
                "environment_variables": {
                    "type": "object",
                    "additionalProperties": {
                        "type": "string"
                    }
                },
                "first_connected_at": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "instance_id": {
                    "type": "string"
                },
                "last_connected_at": {
                    "type": "string"
                },
                "latency": {
                    "description": "DERPLatency is mapped by region name (e.g. \"New York City\", \"Seattle\").",
                    "type": "object",
                    "additionalProperties": {
                        "$ref": "#/definitions/codersdk.DERPRegion"
                    }
                },
                "name": {
                    "type": "string"
                },
                "operating_system": {
                    "type": "string"
                },
                "resource_id": {
                    "type": "string"
                },
                "startup_script": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                },
                "troubleshooting_url": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                },
                "version": {
                    "type": "string"
                }
            }
        },
        "codersdk.WorkspaceApp": {
            "type": "object",
            "properties": {
                "command": {
                    "type": "string"
                },
                "display_name": {
                    "description": "DisplayName is a friendly name for the app.",
                    "type": "string"
                },
                "external": {
                    "description": "External specifies whether the URL should be opened externally on\nthe client or not.",
                    "type": "boolean"
                },
                "health": {
                    "type": "string"
                },
                "healthcheck": {
                    "description": "Healthcheck specifies the configuration for checking app health.",
                    "$ref": "#/definitions/codersdk.Healthcheck"
                },
                "icon": {
                    "description": "Icon is a relative path or external URL that specifies\nan icon to be displayed in the dashboard.",
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "sharing_level": {
                    "type": "string"
                },
                "slug": {
                    "description": "Slug is a unique identifier within the agent.",
                    "type": "string"
                },
                "subdomain": {
                    "description": "Subdomain denotes whether the app should be accessed via a path on the\n` + "`" + `coder server` + "`" + ` or via a hostname-based dev URL. If this is set to true\nand there is no app wildcard configured on the server, the app will not\nbe accessible in the UI.",
                    "type": "boolean"
                },
                "url": {
                    "description": "URL is the address being proxied to inside the workspace.\nIf external is specified, this will be opened on the client.",
                    "type": "string"
                }
            }
        },
        "codersdk.WorkspaceBuild": {
            "type": "object",
            "properties": {
                "build_number": {
                    "type": "integer"
                },
                "created_at": {
                    "type": "string"
                },
                "daily_cost": {
                    "type": "integer"
                },
                "deadline": {
                    "$ref": "#/definitions/codersdk.NullTime"
                },
                "id": {
                    "type": "string"
                },
                "initiator_id": {
                    "type": "string"
                },
                "initiator_name": {
                    "type": "string"
                },
                "job": {
                    "$ref": "#/definitions/codersdk.ProvisionerJob"
                },
                "reason": {
                    "type": "string"
                },
                "resources": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/codersdk.WorkspaceResource"
                    }
                },
                "status": {
                    "type": "string"
                },
                "template_version_id": {
                    "type": "string"
                },
                "template_version_name": {
                    "type": "string"
                },
                "transition": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                },
                "workspace_id": {
                    "type": "string"
                },
                "workspace_name": {
                    "type": "string"
                },
                "workspace_owner_id": {
                    "type": "string"
                },
                "workspace_owner_name": {
                    "type": "string"
                }
            }
        },
        "codersdk.WorkspaceResource": {
            "type": "object",
            "properties": {
                "agents": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/codersdk.WorkspaceAgent"
                    }
                },
                "created_at": {
                    "type": "string"
                },
                "daily_cost": {
                    "type": "integer"
                },
                "hide": {
                    "type": "boolean"
                },
                "icon": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "job_id": {
                    "type": "string"
                },
                "metadata": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/codersdk.WorkspaceResourceMetadata"
                    }
                },
                "name": {
                    "type": "string"
                },
                "type": {
                    "type": "string"
                },
                "workspace_transition": {
                    "type": "string"
                }
            }
        },
        "codersdk.WorkspaceResourceMetadata": {
            "type": "object",
            "properties": {
                "key": {
                    "type": "string"
                },
                "sensitive": {
                    "type": "boolean"
                },
                "value": {
                    "type": "string"
                }
            }
        },
        "codersdk.WorkspacesResponse": {
            "type": "object",
            "properties": {
                "count": {
                    "type": "integer"
                },
                "workspaces": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/codersdk.Workspace"
                    }
                }
            }
        }
    },
    "securityDefinitions": {
        "CoderSessionToken": {
            "type": "apiKey",
            "name": "Coder-Session-Token",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "2.0",
	Host:             "",
	BasePath:         "/api/v2",
	Schemes:          []string{},
	Title:            "Coder API",
	Description:      "Coderd is the service created by running coder server. It is a thin API that connects workspaces, provisioners and users. coderd stores its state in Postgres and is the only service that communicates with Postgres.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
