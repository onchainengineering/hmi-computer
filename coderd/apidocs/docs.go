// Package apidocs GENERATED BY SWAG; DO NOT EDIT
// This file was generated by swaggo/swag
package apidocs

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
    "paths": {}
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "2.0",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "Coderd API",
	Description:      "Coderd is the service created by running coder server. It is a thin API that connects workspaces, provisioners and users. coderd stores its state in Postgres and is the only service that communicates with Postgres.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
