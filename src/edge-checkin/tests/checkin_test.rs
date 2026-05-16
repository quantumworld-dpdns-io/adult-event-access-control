// Placeholder integration test for aev-edge-checkin Spin component.
//
// Full integration tests require a running Spin instance with the
// component deployed. This file serves as a reference for
// the expected test structure once the Wasm module is built.
//
// To run: `spin build && spin up --listen 127.0.0.1:3003`
// Then test with curl or a test harness.

#[test]
fn test_checkin_request_serde() {
    let json = r#"{"ticket_id":"TKT-001","proof":"0xabcd","venue_id":"VENUE-99"}"#;
    let req: serde_json::Value = serde_json::from_str(json).unwrap();
    assert_eq!(req["ticket_id"], "TKT-001");
    assert_eq!(req["proof"], "0xabcd");
    assert_eq!(req["venue_id"], "VENUE-99");
}

#[test]
fn test_verify_request_serde() {
    let json = r#"{"proof":"0xdeadbeef","circuit_type":"age_verify"}"#;
    let req: serde_json::Value = serde_json::from_str(json).unwrap();
    assert_eq!(req["proof"], "0xdeadbeef");
    assert_eq!(req["circuit_type"], "age_verify");
}

#[test]
fn test_health_response_shape() {
    let json = r#"{"status":"ok"}"#;
    let resp: serde_json::Value = serde_json::from_str(json).unwrap();
    assert_eq!(resp["status"], "ok");
}

#[test]
fn test_checkin_response_shape() {
    let json = r#"{"success":true,"message":"Ticket TKT-001 checked in at venue VENUE-99"}"#;
    let resp: serde_json::Value = serde_json::from_str(json).unwrap();
    assert_eq!(resp["success"], true);
    assert!(resp["message"].as_str().unwrap().contains("TKT-001"));
}

#[test]
fn test_verify_response_shape() {
    let json = r#"{"verified":true,"circuit_type":"age_verify"}"#;
    let resp: serde_json::Value = serde_json::from_str(json).unwrap();
    assert_eq!(resp["verified"], true);
    assert_eq!(resp["circuit_type"], "age_verify");
}
