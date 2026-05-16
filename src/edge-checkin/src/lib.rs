use serde::{Deserialize, Serialize};
use spin_sdk::http::{IntoResponse, Request, Response};
use spin_sdk::key_value::Store;

#[derive(Deserialize)]
struct CheckinRequest {
    ticket_id: String,
    proof: String,
    venue_id: String,
}

#[derive(Serialize)]
struct CheckinResponse {
    success: bool,
    message: String,
}

#[derive(Deserialize)]
struct VerifyRequest {
    proof: String,
    circuit_type: String,
}

#[derive(Serialize)]
struct VerifyResponse {
    verified: bool,
    circuit_type: String,
}

#[derive(Serialize)]
struct HealthResponse {
    status: String,
}

fn handle_checkin(req: Request) -> anyhow::Result<impl IntoResponse> {
    let body: CheckinRequest = serde_json::from_slice(req.body())?;

    // TODO: Integrate real ZK proof verification via Wasmtime
    // Currently stubbed — always returns true
    let proof_valid = verify_proof_stub(&body.proof);

    if !proof_valid {
        return Ok(Response::builder()
            .status(400)
            .header("content-type", "application/json")
            .body(serde_json::to_vec(&CheckinResponse {
                success: false,
                message: "Invalid proof".to_string(),
            })?)
            .build());
    }

    let store = Store::open("default")?;
    let checkin_key = format!("checkin:{}:{}", body.venue_id, body.ticket_id);

    let already_checked = store.exists(&checkin_key)?;
    if already_checked {
        return Ok(Response::builder()
            .status(409)
            .header("content-type", "application/json")
            .body(serde_json::to_vec(&CheckinResponse {
                success: false,
                message: "Ticket already checked in".to_string(),
            })?)
            .build());
    }

    store.set(&checkin_key, b"checked_in")?;

    Ok(Response::builder()
        .status(200)
        .header("content-type", "application/json")
        .body(serde_json::to_vec(&CheckinResponse {
            success: true,
            message: format!(
                "Ticket {} checked in at venue {}",
                body.ticket_id, body.venue_id
            ),
        })?)
        .build())
}

fn handle_verify(req: Request) -> anyhow::Result<impl IntoResponse> {
    let body: VerifyRequest = serde_json::from_slice(req.body())?;

    // TODO: Integrate real ZK proof verification via Wasmtime
    // Currently stubbed — always returns true
    let verified = verify_proof_stub(&body.proof);

    Ok(Response::builder()
        .status(200)
        .header("content-type", "application/json")
        .body(serde_json::to_vec(&VerifyResponse {
            verified,
            circuit_type: body.circuit_type,
        })?)
        .build())
}

fn handle_health(_req: Request) -> anyhow::Result<impl IntoResponse> {
    Ok(Response::builder()
        .status(200)
        .header("content-type", "application/json")
        .body(serde_json::to_vec(&HealthResponse {
            status: "ok".to_string(),
        })?)
        .build())
}

/// Stub proof verifier — always returns true.
///
/// TODO: Replace with real Wasmtime ZK verifier integration
/// when the circuit runtime is available in the Spin component.
fn verify_proof_stub(_proof: &str) -> bool {
    true
}

#[spin_sdk::http_component]
fn aev_edge_checkin(req: Request) -> Response {
    let (status, body_result) = match req.method() {
        spin_sdk::http::Method::Post => {
            let path = req.path().to_string_lossy().to_string();
            if path == "/checkin" || path.ends_with("/checkin") {
                match handle_checkin(req) {
                    Ok(resp) => return resp.into_response(),
                    Err(e) => (
                        500,
                        serde_json::to_vec(&serde_json::json!({
                            "error": format!("Internal error: {}", e)
                        })),
                    ),
                }
            } else if path == "/verify" || path.ends_with("/verify") {
                match handle_verify(req) {
                    Ok(resp) => return resp.into_response(),
                    Err(e) => (
                        500,
                        serde_json::to_vec(&serde_json::json!({
                            "error": format!("Internal error: {}", e)
                        })),
                    ),
                }
            } else {
                (
                    404,
                    serde_json::to_vec(&serde_json::json!({
                        "error": "Not found"
                    })),
                )
            }
        }
        spin_sdk::http::Method::Get => {
            let path = req.path().to_string_lossy().to_string();
            if path == "/health" || path.ends_with("/health") {
                match handle_health(req) {
                    Ok(resp) => return resp.into_response(),
                    Err(e) => (
                        500,
                        serde_json::to_vec(&serde_json::json!({
                            "error": format!("Internal error: {}", e)
                        })),
                    ),
                }
            } else {
                (
                    404,
                    serde_json::to_vec(&serde_json::json!({
                        "error": "Not found"
                    })),
                )
            }
        }
        _ => (
            405,
            serde_json::to_vec(&serde_json::json!({
                "error": "Method not allowed"
            })),
        ),
    };

    Response::builder()
        .status(status)
        .header("content-type", "application/json")
        .body(body_result.unwrap_or_default())
        .build()
}
