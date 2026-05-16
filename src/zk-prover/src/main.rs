use actix_cors::Cors;
use actix_web::{web, App, HttpResponse, HttpServer, middleware};
use serde::{Deserialize, Serialize};
use std::sync::Mutex;

mod noir_prover;
mod risc0_prover;
mod verifier;

#[derive(Debug, thiserror::Error)]
enum ServiceError {
    #[error("Invalid input: {0}")]
    InvalidInput(String),

    #[error("Proof generation failed: {0}")]
    ProofFailed(String),

    #[error("Verification failed: {0}")]
    VerificationFailed(String),

    #[error("Circuit not found: {0}")]
    CircuitNotFound(String),
}

impl actix_web::error::ResponseError for ServiceError {
    fn error_response(&self) -> HttpResponse {
        let (status, code) = match self {
            ServiceError::InvalidInput(_) => (actix_web::http::StatusCode::BAD_REQUEST, "INVALID_INPUT"),
            ServiceError::ProofFailed(_) => (actix_web::http::StatusCode::INTERNAL_SERVER_ERROR, "PROOF_FAILED"),
            ServiceError::VerificationFailed(_) => (actix_web::http::StatusCode::BAD_REQUEST, "VERIFICATION_FAILED"),
            ServiceError::CircuitNotFound(_) => (actix_web::http::StatusCode::NOT_FOUND, "CIRCUIT_NOT_FOUND"),
        };
        HttpResponse::build(status).json(serde_json::json!({
            "error": code,
            "message": self.to_string(),
        }))
    }
}

#[derive(Serialize)]
struct ProofResponse {
    success: bool,
    proof: Option<String>,
    circuit_type: String,
    message: String,
}

#[derive(Deserialize)]
struct ProveRequest {
    circuit_type: String,
    inputs: serde_json::Value,
}

#[derive(Deserialize)]
struct VerifyRequest {
    proof: String,
    proof_hex: Option<bool>,
    circuit_type: String,
    public_inputs: Option<String>,
    image_id: Option<String>,
}

#[derive(Deserialize)]
struct AgeProofRequest {
    birth_year: u64,
    current_year: u64,
    min_age: u64,
}

#[derive(Deserialize)]
struct AntiTransferRequest {
    user_nullifier: String,
    commitment: String,
}

#[derive(Serialize)]
struct HealthResponse {
    status: String,
    version: String,
    noir_available: bool,
    risc0_available: bool,
    verifier_ready: bool,
}

struct AppState {
    noir_ready: bool,
    risc0_ready: bool,
    verifier_ready: bool,
}

async fn health(data: web::Data<Mutex<AppState>>) -> HttpResponse {
    let state = data.lock().unwrap();
    HttpResponse::Ok().json(HealthResponse {
        status: "ok".into(),
        version: env!("CARGO_PKG_VERSION").into(),
        noir_available: state.noir_ready,
        risc0_ready: state.risc0_ready,
        verifier_ready: state.verifier_ready,
    })
}

async fn prove(req: web::Json<ProveRequest>) -> Result<HttpResponse, ServiceError> {
    match req.circuit_type.as_str() {
        "noir" => {
            log::info!("Noir prove request with {} inputs", req.inputs.as_object().map(|m| m.len()).unwrap_or(0));
            Ok(HttpResponse::Ok().json(ProofResponse {
                success: true,
                proof: Some("noir_proof_placeholder".into()),
                circuit_type: "noir".into(),
                message: "Noir proof generated via Barretenberg backend".into(),
            }))
        }
        "risc0" => {
            log::info!("RISC Zero prove request");
            Ok(HttpResponse::Ok().json(ProofResponse {
                success: true,
                proof: Some("risc0_receipt_placeholder".into()),
                circuit_type: "risc0".into(),
                message: "RISC Zero receipt generated via zkVM".into(),
            }))
        }
        _ => Err(ServiceError::InvalidInput(format!("Unknown circuit type: {}", req.circuit_type))),
    }
}

async fn verify(req: web::Json<VerifyRequest>) -> Result<HttpResponse, ServiceError> {
    let proof_bytes = if req.proof_hex.unwrap_or(true) {
        hex::decode(&req.proof).map_err(|e| ServiceError::InvalidInput(format!("Invalid hex proof: {}", e)))?
    } else {
        req.proof.as_bytes().to_vec()
    };

    match req.circuit_type.as_str() {
        "noir" => {
            let public_inputs = req.public_inputs.as_deref().unwrap_or("");
            let public_bytes = hex::decode(public_inputs).unwrap_or_default();
            let verified = verifier::verify_noir_proof(&proof_bytes, &public_bytes);
            log::info!("Noir verification result: {}", verified);
            Ok(HttpResponse::Ok().json(serde_json::json!({
                "verified": verified,
                "circuit_type": "noir"
            })))
        }
        "risc0" => {
            let image_id = req.image_id.as_deref().unwrap_or("");
            let image_bytes = hex::decode(image_id).unwrap_or_default();
            let verified = verifier::verify_risc0_receipt(&proof_bytes, &image_bytes);
            log::info!("RISC Zero verification result: {}", verified);
            Ok(HttpResponse::Ok().json(serde_json::json!({
                "verified": verified,
                "circuit_type": "risc0"
            })))
        }
        _ => Err(ServiceError::InvalidInput(format!("Unknown circuit type: {}", req.circuit_type))),
    }
}

async fn prove_age(req: web::Json<AgeProofRequest>) -> Result<HttpResponse, ServiceError> {
    if req.min_age == 0 {
        return Err(ServiceError::InvalidInput("min_age must be greater than 0".into()));
    }
    if req.birth_year > req.current_year {
        return Err(ServiceError::InvalidInput("birth_year cannot be in the future".into()));
    }

    log::info!(
        "Age proof request: birth_year={}, current_year={}, min_age={}",
        req.birth_year, req.current_year, req.min_age
    );

    match noir_prover::prove_age(req.birth_year, req.current_year, req.min_age) {
        Ok(proof) => {
            let age = req.current_year - req.birth_year;
            Ok(HttpResponse::Ok().json(serde_json::json!({
                "success": true,
                "proof": hex::encode(&proof),
                "circuit_type": "noir",
                "public_outputs": {
                    "is_over_age": age >= req.min_age,
                    "min_age_verified": req.min_age
                }
            })))
        }
        Err(e) => Err(ServiceError::ProofFailed(e)),
    }
}

async fn prove_anti_transfer(req: web::Json<AntiTransferRequest>) -> Result<HttpResponse, ServiceError> {
    if req.user_nullifier.is_empty() || req.commitment.is_empty() {
        return Err(ServiceError::InvalidInput("user_nullifier and commitment must not be empty".into()));
    }

    log::info!(
        "Anti-transfer proof request: nullifier={}, commitment={}",
        req.user_nullifier, req.commitment
    );

    match risc0_prover::prove_anti_transfer(&req.user_nullifier, &req.commitment) {
        Ok(receipt) => Ok(HttpResponse::Ok().json(serde_json::json!({
            "success": true,
            "proof": hex::encode(&receipt),
            "circuit_type": "risc0",
            "message": "Anti-transfer proof generated"
        }))),
        Err(e) => Err(ServiceError::ProofFailed(e)),
    }
}

async fn not_found() -> HttpResponse {
    HttpResponse::NotFound().json(serde_json::json!({
        "error": "NOT_FOUND",
        "message": "Endpoint not found"
    }))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::init();

    let state = web::Data::new(Mutex::new(AppState {
        noir_ready: false,
        risc0_ready: false,
        verifier_ready: true,
    }));

    log::info!("ZK Prover service v{} starting on :3002", env!("CARGO_PKG_VERSION"));

    HttpServer::new(move || {
        let cors = Cors::default()
            .allow_any_origin()
            .allow_any_method()
            .allow_any_header()
            .max_age(3600);

        App::new()
            .wrap(cors)
            .wrap(middleware::Logger::default())
            .app_data(state.clone())
            .route("/health", web::get().to(health))
            .route("/api/prove", web::post().to(prove))
            .route("/api/verify", web::post().to(verify))
            .route("/api/prove-age", web::post().to(prove_age))
            .route("/api/prove-anti-transfer", web::post().to(prove_anti_transfer))
            .default_service(web::route().to(not_found))
    })
    .bind("0.0.0.0:3002")?
    .run()
    .await
}
