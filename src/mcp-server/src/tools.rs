use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::collections::HashMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Tool {
    pub name: String,
    pub description: String,
    pub input_schema: Value,
}

/// Returns the full list of MCP tools exposed by this server.
pub fn list_tools() -> Vec<Tool> {
    vec![
        Tool {
            name: "list_events".into(),
            description: "List all events from the backend".into(),
            input_schema: serde_json::json!({
                "type": "object",
                "properties": {}
            }),
        },
        Tool {
            name: "get_event".into(),
            description: "Get details for a specific event by ID".into(),
            input_schema: serde_json::json!({
                "type": "object",
                "properties": {
                    "id": {
                        "type": "string",
                        "description": "Event ID"
                    }
                },
                "required": ["id"]
            }),
        },
        Tool {
            name: "get_tickets".into(),
            description: "List all tickets from the backend".into(),
            input_schema: serde_json::json!({
                "type": "object",
                "properties": {}
            }),
        },
        Tool {
            name: "check_analytics".into(),
            description: "Get analytics overview from the backend".into(),
            input_schema: serde_json::json!({
                "type": "object",
                "properties": {}
            }),
        },
        Tool {
            name: "get_venue_heatmap".into(),
            description: "Get venue heatmap data for an event".into(),
            input_schema: serde_json::json!({
                "type": "object",
                "properties": {
                    "id": {
                        "type": "string",
                        "description": "Event ID"
                    }
                },
                "required": ["id"]
            }),
        },
        Tool {
            name: "create_alert".into(),
            description: "Create a new alert for an event".into(),
            input_schema: serde_json::json!({
                "type": "object",
                "properties": {
                    "id": {
                        "type": "string",
                        "description": "Event ID"
                    },
                    "message": {
                        "type": "string",
                        "description": "Alert message"
                    },
                    "severity": {
                        "type": "string",
                        "description": "Alert severity (info, warning, critical)"
                    }
                },
                "required": ["id", "message"]
            }),
        },
        Tool {
            name: "recommend_events".into(),
            description: "Get event recommendations from the backend".into(),
            input_schema: serde_json::json!({
                "type": "object",
                "properties": {}
            }),
        },
    ]
}

/// Maps a tool name to its backend URL path.
fn tool_backend_path(name: &str) -> Option<&'static str> {
    match name {
        "list_events" => Some("/api/events"),
        "get_event" => Some("/api/events/"),
        "get_tickets" => Some("/api/tickets"),
        "check_analytics" => Some("/api/analytics/overview"),
        "get_venue_heatmap" => Some("/api/events/"),
        "create_alert" => Some("/api/events/"),
        "recommend_events" => Some("/api/recommend/events"),
        _ => None,
    }
}

fn tool_http_method(name: &str) -> &'static str {
    match name {
        "create_alert" => "POST",
        _ => "GET",
    }
}

/// Attempts to extract an `id` parameter from the JSON-RPC params.
fn extract_id(params: &Value) -> Option<String> {
    params.get("id").and_then(|v| v.as_str()).map(|s| s.to_string())
}

/// Resolves the full backend URL for a given tool invocation.
fn build_backend_url(base: &str, name: &str, params: &Value) -> Option<String> {
    let path = tool_backend_path(name)?;
    match name {
        "get_event" | "get_venue_heatmap" | "create_alert" => {
            let id = extract_id(params)?;
            Some(format!("{}{}{}", base, path, id))
        }
        _ => Some(format!("{}{}", base, path)),
    }
}

/// Calls a tool by name against the backend and returns the JSON result.
pub async fn call_tool(name: &str, params: Value, backend_url: &str) -> Result<Value, String> {
    let tools = list_tools();
    if !tools.iter().any(|t| t.name == name) {
        return Err(format!("Unknown tool: {}", name));
    }

    let url = build_backend_url(backend_url, name, &params)
        .ok_or_else(|| format!("Failed to build URL for tool: {}", name))?;

    let client = reqwest::Client::new();
    let method = tool_http_method(name);

    let resp = match method {
        "POST" => {
            let body = serde_json::to_value(&params).map_err(|e| e.to_string())?;
            client.post(&url).json(&body).send().await
        }
        _ => client.get(&url).send().await,
    };

    match resp {
        Ok(r) => {
            let status = r.status();
            let body: Value = r.json().await.map_err(|e| e.to_string())?;
            let mut result = serde_json::Map::new();
            result.insert("status".into(), Value::Number(status.as_u16().into()));
            result.insert("data".into(), body);
            Ok(Value::Object(result))
        }
        Err(e) => Err(format!("Request failed: {}", e)),
    }
}
