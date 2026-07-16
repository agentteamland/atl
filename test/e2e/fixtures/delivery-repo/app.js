// Trivial app surface for the delivery-team GitHub-backend e2e fixture.
// A developer worker adds/extends a small pure function here; the tester runs
// app.test.js. Kept deliberately tiny so the loop's proof is the delivery
// mechanics (issue -> PR -> merge to dev -> close), not the app.

function add(a, b) {
  return a + b;
}

module.exports = { add };
