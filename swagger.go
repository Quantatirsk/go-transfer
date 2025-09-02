package main

import (
	"encoding/json"
	"net/http"
)

// generateSwaggerJSON 生成Swagger JSON文档
func generateSwaggerJSON(host string) string {
	doc := map[string]interface{}{
		"swagger": "2.0",
		"info": map[string]interface{}{
			"version":     "2.0.0",
			"title":       "go-transfer API",
			"description": "纯流式文件传输服务 - 零缓存，支持超大文件",
		},
		"host":     host,
		"basePath": "/",
		"schemes":  []string{"http", "https"},
		"paths": map[string]interface{}{
			"/upload": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "上传文件",
					"description": "支持浏览器FormData和命令行二进制流，零缓存传输",
					"consumes":    []string{"multipart/form-data", "application/octet-stream"},
					"produces":    []string{"text/plain"},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "query",
							"description": "文件名（可选，FormData时自动获取）",
							"required":    false,
							"type":        "string",
						},
						{
							"name":        "file",
							"in":          "formData",
							"description": "选择要上传的文件",
							"required":    true,
							"type":        "file",
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "上传成功",
							"examples": map[string]interface{}{
								"text/plain": "文件上传成功: example.zip (1024 bytes)",
							},
						},
						"400": map[string]interface{}{
							"description": "缺少文件名参数",
						},
						"500": map[string]interface{}{
							"description": "服务器错误",
						},
					},
				},
			},
			"/status": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "服务状态",
					"description": "获取服务运行状态和健康检查",
					"produces":    []string{"application/json"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "状态信息",
							"schema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"status": map[string]interface{}{
										"type":        "string",
										"description": "健康状态",
										"example":     "ok",
									},
									"mode": map[string]interface{}{
										"type":        "string",
										"description": "运行模式",
										"enum":        []string{"receiver", "relay", "gateway"},
									},
									"port": map[string]interface{}{
										"type":        "integer",
										"description": "监听端口",
										"example":     17002,
									},
									"timestamp": map[string]interface{}{
										"type":        "integer",
										"description": "时间戳",
									},
									"version": map[string]interface{}{
										"type":        "string",
										"description": "版本号",
										"example":     "2.0.0",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	jsonData, _ := json.MarshalIndent(doc, "", "  ")
	return string(jsonData)
}

// handleSwaggerJSON 处理Swagger JSON请求
func handleSwaggerJSON(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if host == "" {
		host = "localhost:17002"
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(generateSwaggerJSON(host)))
}

// handleSwaggerUI 处理Swagger UI请求
func handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <title>go-transfer API文档</title>
    <link rel="stylesheet" type="text/css" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin:0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
    window.onload = function() {
        window.ui = SwaggerUIBundle({
            url: "/swagger.json",
            dom_id: '#swagger-ui',
            deepLinking: true,
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIStandalonePreset
            ],
            plugins: [
                SwaggerUIBundle.plugins.DownloadUrl
            ],
            layout: "StandaloneLayout",
            validatorUrl: null,
            tryItOutEnabled: true
        })
    }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}
