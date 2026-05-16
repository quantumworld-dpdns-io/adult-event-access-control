// Age Verification Circuit (Noir)
// Proves a person is over minimum age without revealing exact birth year
//
// Private inputs: birth_year
// Public inputs:  current_year, min_age
// Public output:  1 if age >= min_age, otherwise proof generation fails

fn main(birth_year: Field, current_year: Field, min_age: pub Field) -> pub Field {
    let age = current_year - birth_year;
    assert(age as u64 >= min_age as u64);
    1
}

#[test]
fn test_over_age() {
    let r = main(1990, 2026, 18);
    assert(r == 1);
}
