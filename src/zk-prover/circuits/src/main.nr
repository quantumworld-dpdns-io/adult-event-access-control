// Age Verification Circuit (Noir)
// Proves that a person is over a minimum age without revealing their exact birthdate
//
// Private inputs:
//   - birthdate_hash: hashed representation of the user's birthdate
//   - current_year: the current year
//
// Public inputs:
//   - min_age: the minimum age required
//   - is_over_age: boolean result (publicly verifiable)
//
// Constraint: current_year - birth_year >= min_age

fn main(
    birth_year: Field,      // Private: user's birth year
    current_year: Field,    // Private: current year (could be public)
    min_age: pub Field      // Public: the minimum age requirement
) -> pub bool {
    let age = current_year - birth_year;
    let is_over_age = age >= min_age;
    assert(is_over_age);
    is_over_age
}

#[test]
fn test_age_proof_valid() {
    let result = main(1990, 2026, 18);
    assert(result == true);
}

#[test]
fn test_age_proof_invalid() {
    let result = main(2015, 2026, 18);
    assert(result == false);
}
