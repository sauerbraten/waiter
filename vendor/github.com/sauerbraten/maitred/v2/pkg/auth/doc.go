// package auth implements Sauerbraten's player authentication mechanism.
//
// The mechanism relies on the associativity of scalar multiplication on elliptic curves: private keys
// are random (big) scalars, and the corresponding public key is created by multiplying the curve base point
// with the private key. (This means the public key is another point on the curve.)
// To check for posession of the private key belonging to a public key known to the server, the base point is
// multiplied with another random, big scalar (the "secret") and the resulting point is sent to the user as
// "challenge". The user multiplies the challenge curve point with his private key (a scalar), and sends the
// X coordinate of the resulting point back to the server.
// The server instead multiplies the user's public key with the secret scalar. Since pub = base * priv,
// pub * secret = (base * priv) * secret = (base * secret) * priv = challenge * priv. Because of the curve's
// symmetry, there are exactly two points on the curve at any given X. For simplicity (and maybe performance),
// the server is satisfied when the user responds with the correct X.
package auth
