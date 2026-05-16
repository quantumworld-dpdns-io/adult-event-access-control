mod tools;

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::env;
use std::net::SocketAddr;
use tokio::io::{AsyncBufReadExt, AsyncWriteExt, BufReader};
use tokio::net::{TcpListener, TcpStream};

#[derive(Serialize, Deserialize)]
struct JsonRpcRequest {
    jsonrpc: String,
    method: String,
    #[serde(default)]
    params: serde_json::Value,
    id: u64,
}

#[derive(Serialize)]
struct JsonRpcResponse {
    jsonrpc: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    result: Option<serde_json::Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<JsonRpcError>,
    id: u64,
}

#[derive(Serialize)]
struct JsonRpcError {
    code: i64,
    message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    data: Option<serde_json::Value>,
}

impl JsonRpcResponse {
    fn success(id: u64, result: serde_json::Value) -> Self {
        JsonRpcResponse {
            jsonrpc: "2.0".into(),
            result: Some(result),
            error: None,
            id,
        }
    }

    fn error(id: u64, code: i64, message: String) -> Self {
        JsonRpcResponse {
            jsonrpc: "2.0".into(),
            result: None,
            error: Some(JsonRpcError {
                code,
                message,
                data: None,
            }),
            id,
        }
    }
}

#[derive(Serialize)]
struct ToolDescription {
    name: String,
    description: String,
    input_schema: serde_json::Value,
}

#[derive(Serialize)]
struct ResourceDescription {
    uri: String,
    name: String,
    description: String,
    mime_type: String,
}

const BACKEND_PORT_ENV: &str = "BACKEND_URL";
const DEFAULT_BACKEND_URL: &str = "http://localhost:8080";
const LISTEN_ADDR: &str = "127.0.0.1:3100";

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let backend_url =
        env::var(BACKEND_PORT_ENV).unwrap_or_else(|_| DEFAULT_BACKEND_URL.to_string());

    eprintln!("aev-mcp-server starting on {}", LISTEN_ADDR);
    eprintln!("backend URL: {}", backend_url);

    let listener = TcpListener::bind(LISTEN_ADDR).await?;

    loop {
        let (stream, addr) = listener.accept().await?;
        eprintln!("connection from {}", addr);
        let backend = backend_url.clone();
        tokio::spawn(async move {
            if let Err(e) = handle_connection(stream, &backend).await {
                eprintln!("error handling {}: {}", addr, e);
            }
        });
    }
}

async fn handle_connection(stream: TcpStream, backend_url: &str) -> anyhow::Result<()> {
    let (reader, mut writer) = stream.into_split();
    let mut buf_reader = BufReader::new(reader);
    let mut line = String::new();

    // Read one line — a single JSON-RPC request per connection
    buf_reader.read_line(&mut line).await?;
    let line = line.trim();

    if line.is_empty() {
        return Ok(());
    }

    eprintln!("-> {}", line);

    let response = match serde_json::from_str::<JsonRpcRequest>(line) {
        Ok(req) => handle_rpc(req, backend_url).await,
        Err(_) => JsonRpcResponse::error(0, -32700, "Parse error".into()),
    };

    let resp_json = serde_json::to_string(&response)?;
    eprintln!("<- {}", resp_json);

    writer.write_all(resp_json.as_bytes()).await?;
    writer.write_all(b"\n").await?;

    Ok(())
}

async fn handle_rpc(req: JsonRpcRequest, backend_url: &str) -> JsonRpcResponse {
    match req.method.as_str() {
        "tools/list" => {
            let tool_list: Vec<ToolDescription> = tools::list_tools()
                .into_iter()
                .map(|t| ToolDescription {
                    name: t.name,
                    description: t.description,
                    input_schema: t.input_schema,
                })
                .collect();
            JsonRpcResponse::success(req.id, serde_json::json!({ "tools": tool_list }))
        }
        "tools/call" => {
            let name = req.params.get("name").and_then(|v| v.as_str());
            let arguments = req.params.get("arguments").cloned().unwrap_or_default();
            match name {
                Some(n) => match tools::call_tool(n, arguments, backend_url).await {
                    Ok(result) => {
                        JsonRpcResponse::success(req.id, serde_json::json!({ "content": [{"type": "text", "text": result.to_string()}], "isError": false }))
                    }
                    Err(e) => JsonRpcResponse::error(req.id, -32000, e),
                },
                None => JsonRpcResponse::error(req.id, -32602, "Missing tool name".into()),
            }
        }
        "resources/list" => {
            let resources = vec![
                ResourceDescription {
                    uri: "aev://events".into(),
                    name: "All Events".into(),
                    description: "List of all events from the backend".into(),
                    mime_type: "application/json".into(),
                },
                ResourceDescription {
                    uri: "aev://analytics".into(),
                    name: "Analytics Overview".into(),
                    description: "Analytics overview data".into(),
                    mime_type: "application/json".into(),
                },
                ResourceDescription {
                    uri: "aev://tickets".into(),
                    name: "All Tickets".into(),
                    description: "List of all tickets".into(),
                    mime_type: "application/json".into(),
                },
            ];
            JsonRpcResponse::success(req.id, serde_json::json!({ "resources": resources }))
        }
        "resources/read" => {
            let uri = req.params.get("uri").and_then(|v| v.as_str());
            match uri {
                Some(u) => {
                    let path = match u {
                        "aev://events" => "/api/events",
                        "aev://analytics" => "/api/analytics/overview",
                        "aev://tickets" => "/api/tickets",
                        _ => {
                            return JsonRpcResponse::error(
                                req.id,
                                -32602,
                                format!("Unknown resource: {}", u),
                            )
                        }
                    };
                    let url = format!("{}{}", backend_url, path);
                    match reqwest::get(&url).await {
                        Ok(resp) => {
                            let body: serde_json::Value =
                                resp.json().await.unwrap_or(serde_json::json!(null));
                            JsonRpcResponse::success(
                                req.id,
                                serde_json::json!({
                                    "contents": [{
                                        "uri": u,
                                        "mimeType": "application/json",
                                        "text": body.to_string()
                                    }]
                                }),
                            )
                        }
                        Err(e) => {
                            JsonRpcResponse::error(req.id, -32000, format!("Backend error: {}", e))
                        }
                    }
                }
                None => JsonRpcResponse::error(req.id, -32602, "Missing resource URI".into()),
            }
        }
        _ => JsonRpcResponse::error(
            req.id,
            -32601,
            format!("Method not found: {}", req.method),
        ),
    }
}
