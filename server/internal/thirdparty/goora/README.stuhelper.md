# StuHelper go-ora policy fork

This directory contains the root Go package from `github.com/sijms/go-ora/v2`
v2.9.0. Its unchanged subpackages continue to come from the checksum-pinned Go
module. The upstream license is preserved in `LICENSE`.

StuHelper carries this small source fork because upstream automatically sends an
`AUTH_ALTER_SESSION` command during authentication and falls back to a PL/SQL
block when loading NLS metadata. Both operations violate the BUAA Oracle policy,
which permits only authentication and locally fixed `SELECT` statements.

The intentional differences from v2.9.0 are limited to:

- omit the authentication-time session mutation payload;
- load fallback NLS metadata with one fixed `SELECT`;
- reject the session-parameter helper instead of executing session mutations;
- reject transactions and all `Exec` paths, and validate every query as one
  side-effect-free `SELECT` before it reaches the wire;
- keep Oracle auto-commit flags disabled so cursor execution cannot emit a
  commit operation, even when no data has changed;
- require the Oracle 12c+ PBKDF2/SHA-512 password verifier and fail closed
  instead of falling back to 10g DES or 11g SHA-1 authentication;
- reject out-of-range session identifiers and numeric scan destinations instead
  of silently narrowing attacker- or server-controlled integers.

Do not merge an upstream update mechanically. Re-import the root non-test Go
files, reapply these reviewed changes, retain the license, and run the policy test,
the external-data tests, full lint, race tests, and build before use.

The root package also inherits Oracle wire-compatibility primitives and integer
encodings which generic application-code scanners report as protocol-level
findings. `make security` therefore does not present this pinned third-party
package as first-party application code. Before scanning first-party packages,
it runs `scripts/check-goora-fork.sh`, which requires every unmodified source
file to remain byte-for-byte identical to v2.9.0 and every reviewed policy-fork
file to retain its approved hash. The runtime SELECT-only policy and its tests
remain part of the normal Go test suite.
