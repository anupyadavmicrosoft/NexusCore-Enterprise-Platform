package docs

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const OpenAPISpecJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "NexusCore Enterprise API Gateway",
    "description": "High-throughput secure ingress proxy and routing namespace for NexusCore",
    "version": "1.0.0",
    "contact": {
      "name": "NexusCore Platform Engineers",
      "email": "architecture@nexuscore.internal"
    }
  },
  "servers": [
    {
      "url": "http://localhost:8080",
      "description": "Local Ingress Gateway"
    }
  ],
  "paths": {
    "/healthz": {
      "get": {
        "summary": "Liveness Probe Check",
        "description": "Validates if the container is functional and accepting connections.",
        "responses": {
          "200": {
            "description": "Healthy",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "status": { "type": "string" },
                    "service": { "type": "string" },
                    "timestamp": { "type": "string" }
                  }
                }
              }
            }
          }
        }
      }
    },
    "/readyz": {
      "get": {
        "summary": "Readiness Probe Check",
        "description": "Checks the connectivity of all downstream microservices before accepting cluster traffic.",
        "responses": {
          "200": {
            "description": "Ready to receive traffic"
          },
          "503": {
            "description": "One or more dependencies are unreachable"
          }
        }
      }
    },
    "/api/v1/auth/login": {
      "post": {
        "summary": "Client and Admin Login",
        "description": "Authenticates credentials and issues secure bearer tokens.",
        "parameters": [
          {
            "name": "X-API-Key",
            "in": "header",
            "required": false,
            "schema": { "type": "string" }
          }
        ],
        "responses": {
          "200": { "description": "Successfully authenticated" },
          "401": { "description": "Invalid key or password" }
        }
      }
    },
    "/api/v1/compute/transactions": {
      "get": {
        "summary": "Query Compute Transactions",
        "description": "Retrieves the historical record of compute engine logs. Requires Admin or Operator clearance.",
        "security": [
          { "BearerAuth": [] }
        ],
        "responses": {
          "200": { "description": "Successful operation" },
          "401": { "description": "Invalid bearer token" },
          "403": { "description": "Insufficient roles" }
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT"
      },
      "APIKeyAuth": {
        "type": "apiKey",
        "in": "header",
        "name": "X-API-Key"
      }
    }
  }
}`

func RegisterSwaggerPortal(r *gin.Engine) {
	// JSON Endpoint
	r.GET("/docs/openapi.json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, OpenAPISpecJSON)
	})

	// Beautiful Swagger UI wrapper HTML page
	r.GET("/swagger", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>NexusCore API Swagger Portal</title>
    <link rel="stylesheet" type="text/css" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.9.0/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.9.0/swagger-ui-bundle.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.9.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            window.ui = SwaggerUIBundle({
                url: "/docs/openapi.json",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`
		c.String(http.StatusOK, html)
	})
}
