#!/bin/sh
# Creates the self-signed code-signing identity "simple-ideas Self-Signed" in
# your login keychain, once. build-mac.sh signs si Studio.app with it, so the
# app keeps one identity across updates and macOS keeps the permissions it
# granted. It does not make the app "from an identified developer": the first
# open of a downloaded copy still needs right-click ▸ Open (docs/desktop.md).
# The key never leaves your keychain; nothing here is committed.
set -eu
NAME=${SI_SIGN_IDENTITY:-simple-ideas Self-Signed}
if security find-certificate -c "$NAME" >/dev/null 2>&1; then
	echo "\"$NAME\" is already in your keychain"; exit 0
fi
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
cat > "$TMP/cert.cnf" <<CNF
[req]
distinguished_name = dn
x509_extensions = ext
prompt = no
[dn]
CN = $NAME
[ext]
keyUsage = critical, digitalSignature
extendedKeyUsage = critical, codeSigning
basicConstraints = critical, CA:false
CNF
openssl req -x509 -newkey rsa:2048 -nodes -days 3650 -keyout "$TMP/key.pem" -out "$TMP/cert.pem" -config "$TMP/cert.cnf" 2>/dev/null
PASS=$(openssl rand -hex 16)
openssl pkcs12 -export -inkey "$TMP/key.pem" -in "$TMP/cert.pem" -name "$NAME" -out "$TMP/id.p12" -passout "pass:$PASS" -legacy 2>/dev/null ||
	openssl pkcs12 -export -inkey "$TMP/key.pem" -in "$TMP/cert.pem" -name "$NAME" -out "$TMP/id.p12" -passout "pass:$PASS"
security import "$TMP/id.p12" -k "$HOME/Library/Keychains/login.keychain-db" -P "$PASS" -T /usr/bin/codesign
echo "created \"$NAME\"; make desktop-mac now signs with it"
