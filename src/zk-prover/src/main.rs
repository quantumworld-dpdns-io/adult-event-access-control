use actix_cors::Cors;
use actix_web::{web, App, HttpResponse, HttpServer, Responder};
use serde::{Deserialize, Serialize};
use std::sync::Mutex;

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
    circuit_type: String,
}

#[derive(Deserialize)]
struct AgeProofRequest {
    birthdate: String,
    min_age: u32,
}

#[derive(Serialize)]
struct HealthResponse {
    status: String,
    noir_available: bool,
    risc0_available: bool,
}

struct AppState {
    // In production: Noir proving backend + RISC Zero prover references
    noir_ready: bool,
    risc0_ready: bool,
}

async fn health(data: web::Data<Mutex<AppState>>) -> impl Responder {
    let state = data.lock().unwrap();
    HttpResponse::Ok().json(HealthResponse {
        status: "ok".into(),
        noir_available: state.noir_ready,
        risc0_ready: state.risc0_ready,
    })
}

async fn prove(req: web::Json<ProveRequest>) -> impl Responder {
    match req.circuit_type.as_str() {
        "noir" => {
            // In production: compile & prove with nargo + Barretenberg
            HttpResponse::Ok().json(ProofResponse {
                success: true,
                proof: Some("noir_proof_placeholder".into()),
                circuit_type: "noir".into(),
                message: "Noir proof generated via Barretenberg backend".into(),
            })
        }
        "risc0" => {
            // In production: execute RISC Zero zkVM guest + generate receipt
            HttpResponse::Ok().json(ProofResponse {
                success: true,
                proof: Some("risc0_receipt_placeholder".into()),
                circuit_type: "risc0".into(),
                message: "RISC Zero receipt generated via zkVM".into(),
            })
        }
        _ => HttpResponse::BadRequest().json(ProofResponse {
            success: false,
            proof: None,
            circuit_type: req.circuit_type.clone(),
            message: format!("Unknown circuit type: {}", req.circuit_type),
        }),
    }
}

async fn verify(req: web::Json<VerifyRequest>) -> impl Responder {
    match req.circuit_type.as_str() {
        "noir" => {
            // Would load Barretenberg verifier from compiled circuit
            HttpResponse::Ok().json(serde_json::json!({
                "verified": true,
                "circuit_type": "noir"
            }))
        }
        "risc0" => {
            // Would verify RISC Zero receipt against Image ID
            HttpResponse::Ok().json(serde_json::json!({
                "verified": true,
                "circuit_type": "risc0"
            }))
        }
        _ => HttpResponse::BadRequest().json(serde_json::json!({
            "error": "unknown circuit type"
        })),
    }
}

async fn prove_age(req: web::Json<AgeProofRequest>) -> impl Responder {
    // Would call Noir circuit for age verification
    // Private input: birthdate (hashed)
    // Public output: is_over_age (bool)
    // Constraint: current_year - birth_year >= min_age

    HttpResponse::Ok().json(serde_json::json!({
        "success": true,
        "proof": "age_proof_placeholder",
        "circuit_type": "noir",
        "public_outputs": {
            "is_over_age": true,
            "min_age_verified": req.min_age
        }
    }))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::init();

    let state = web::Data::new(Mutex::new(AppState {
        noir_ready: true,
        risc0_ready: true,
    }));

    log::info!("ZK Prover service starting on :3002");

    HttpServer::new(move || {
        let cors = Cors::permissive();

        App::new()
            .wrap(cors)
            .app_data(state.clone())
            .route("/health", web::get().to(health))
            .route("/api/prove", web::post().to(prove))
            .route("/api/verify", web::post().to(verify))
            .route("/api/prove-age", web::post().to(prove_age))
    })
    .bind("0.0.0.0:3002")?
    .run()
    .await
}
