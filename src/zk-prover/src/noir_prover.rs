use log::info;
use std::collections::HashMap;

/// A Noir field element represented as 32 big-endian bytes.
///
/// In Noir, a `Field` is an element of the base field of the proving
/// backend's elliptic curve (BN254 for Barretenberg). Values are
/// stored as 32-byte big-endian unsigned integers.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Field(pub [u8; 32]);

impl Field {
    /// Creates a field element from a `u64` value.
    pub fn from_u64(value: u64) -> Self {
        let mut bytes = [0u8; 32];
        bytes[24..32].copy_from_slice(&value.to_be_bytes());
        Field(bytes)
    }

    /// Creates a field element from a hex string.
    pub fn from_hex(hex: &str) -> Result<Self, String> {
        let mut bytes = [0u8; 32];
        let decoded = hex::decode(hex).map_err(|e| format!("Invalid hex: {}", e))?;
        if decoded.len() > 32 {
            return Err("Hex value too large for field element".into());
        }
        bytes[32 - decoded.len()..].copy_from_slice(&decoded);
        Ok(Field(bytes))
    }
}

/// Generates a zero-knowledge proof that a person's age meets a minimum threshold.
///
/// # Circuit Architecture
///
/// The Noir circuit (`circuits/src/main.nr`) implements:
///
/// ```noir
/// fn main(birth_year: Field, current_year: Field, min_age: Field) {
///     let age = current_year - birth_year;
///     constrain age >= min_age;
/// }
/// ```
///
/// `birth_year` is a **private** witness — it is never revealed.
/// `current_year` and `min_age` are **public** inputs.
/// The proof attests: "I know a `birth_year` such that `current_year - birth_year >= min_age`"
/// without revealing the actual birth year.
///
/// # Arguments
/// * `birth_year` - The prover's birth year (private input)
/// * `current_year` - The current year (public input)
/// * `min_age` - The minimum age requirement (public input)
///
/// # Returns
/// Serialized proof bytes on success.
pub fn prove_age(birth_year: u64, current_year: u64, min_age: u64) -> Result<Vec<u8>, String> {
    info!(
        "Noir prove_age: birth_year={}, current_year={}, min_age={}",
        birth_year, current_year, min_age
    );

    // TODO: Real implementation:
    //
    // 1. Compile circuit (if not already compiled):
    //    $ cd circuits && nargo compile age_test
    //
    //    This generates circuits/target/age_test.json containing the ACIR
    //    representation and the ABI (ABI = Application Binary Interface
    //    describing the expected inputs/outputs).
    //
    // 2. Serialize inputs into Prover.toml format:
    //
    //    birth_year = "1947"   # private
    //    current_year = "2025" # public
    //    min_age = "21"        # public
    //
    // 3. Execute the prover via `bb` (Barretenberg) or `nargo prove`:
    //
    //    $ nargo prove proof_age_test
    //
    //    Or using bb directly:
    //    $ bb prove -c circuits/target/age_test.json \
    //               -w circuits/target/age_test.gz \
    //               -o proof_age_test.proof
    //
    // 4. Read back the proof file and return the bytes.
    //
    // For now, return a 64-byte placeholder proof.

    Ok(vec![0u8; 64])
}

/// Generates a zero-knowledge proof for an arbitrary Noir circuit.
///
/// # Architecture
///
/// Noir circuits are compiled by `nargo` into two artifacts:
/// - **ACIR representation** (`target/<circuit>.json`): the constraint system
/// - **ABI** (embedded in the JSON): input/output type definitions
///
/// The prover:
/// 1. Reads the compiled circuit JSON
/// 2. Constructs a witness assignment from the provided `inputs` map
/// 3. Calls the proving backend (Barretenberg via `bb`) to generate a proof
///
/// # Arguments
/// * `circuit_path` - Path to the compiled circuit JSON (output of `nargo compile`)
/// * `inputs` - Witness name → Field value mapping matching the circuit's ABI
///
/// # Returns
/// Serialized proof bytes on success.
pub fn prove_custom(circuit_path: &str, inputs: &HashMap<String, Field>) -> Result<Vec<u8>, String> {
    info!(
        "Noir prove_custom: circuit_path={}, input_count={}",
        circuit_path,
        inputs.len()
    );

    if inputs.is_empty() {
        return Err("No inputs provided for circuit".into());
    }

    // TODO: Real implementation:
    //
    // 1. Read and parse the compiled circuit JSON:
    //
    //    let circuit_json = std::fs::read_to_string(circuit_path)
    //        .map_err(|e| format!("Failed to read circuit: {}", e))?;
    //    let circuit: NoirCircuit = serde_json::from_str(&circuit_json)
    //        .map_err(|e| format!("Failed to parse circuit: {}", e))?;
    //
    // 2. Build a Prover.toml from the inputs HashMap:
    //
    //    let mut prover_toml = String::new();
    //    for (name, field) in inputs {
    //        writeln!(prover_toml, "{} = \"{}\"", name, hex::encode(&field.0))?;
    //    }
    //    std::fs::write("Prover.toml", prover_toml)?;
    //
    // 3. Execute Barretenberg prove:
    //
    //    use std::process::Command;
    //    let output = Command::new("bb")
    //        .args(&["prove", "-c", circuit_path, "-w", "Prover.toml", "-o", "proof.bin"])
    //        .output()
    //        .map_err(|e| format!("Failed to execute bb: {}", e))?;
    //
    //    if !output.status.success() {
    //        return Err(format!("bb failed: {}", String::from_utf8_lossy(&output.stderr)));
    //    }
    //
    // 4. Read the proof file:
    //
    //    let proof = std::fs::read("proof.bin")
    //        .map_err(|e| format!("Failed to read proof: {}", e))?;
    //    Ok(proof)

    Ok(vec![0u8; 64])
}
