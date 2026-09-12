# Security Policy

## Supported Versions

We actively maintain and provide security patches for the following versions of **1stcrack**:

| Version | Supported |
|---|---|
| 1.0.x | :white_check_mark: |

---

## Reporting a Vulnerability

We take the integrity and reliability of roastery operational data seriously. If you discover a security vulnerability, please report it privately.

**Do not open a public GitHub issue for security vulnerabilities.**

Please report all vulnerability details via email to **dzulkiflianwar2@gmail.com**.

### What to Include in Your Report

To help us investigate and remediate the issue promptly, please include:
- A clear description of the vulnerability.
- Steps to reproduce the issue (including sample CLI commands, test data, or proof-of-concept scripts).
- Potential impact on data integrity, financial calculation correctness, or process isolation.

### Areas of Critical Interest

- **Transaction Atomicity & Isolation**: Race conditions or transaction leaks that could cause inconsistent stock balances or double-spending of inventory.
- **Input Sanitization & Storage**: Injection vulnerabilities in SQL queries or unsafe handling of file paths during receipt export (`receipts/` directory traversal).
- **Arithmetic Overflow**: Edge cases where extreme weights or monetary amounts could cause integer overflow in `int64` calculations.

### Our Response Process

1. **Acknowledgment**: We will acknowledge receipt of your report within 48 hours.
2. **Investigation**: We will verify the vulnerability and evaluate its severity.
3. **Remediation**: We will develop and test a fix in a private branch.
4. **Disclosure**: We will publish a patched release and coordinate public disclosure.
