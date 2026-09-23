# Security Policy

## Supported Versions

We release patches and security updates for the current minor release of `llm-router`.

| Version | Supported          |
| ------- | ------------------ |
| v0.x    | :white_check_mark: |

---

## Secret & Credential Handling

`llm-router` handles LLM credentials (`Route.APIKey`):
* **Memory-only**: `llm-router` never writes API keys to disk, databases, or external log sinks.
* **No network transmission**: The library never makes network calls and does not transmit keys over the internet.
* **No leaks in errors**: Sentinel and wrapped error messages never include or format `APIKey` or other credential data.

When writing applications, tests, or documentation:
* Do not commit real API keys to repositories. Use environment variables (e.g. `os.Getenv`) or secret managers.
* In tests and examples, use obvious dummy keys (e.g. `"mock-key"`, `"test-secret"`).

---

## Reporting a Vulnerability

If you discover a potential security vulnerability in `llm-router`, please do **not** open a public issue.

Instead, please report it privately:
* Via GitHub Security Advisory: Navigate to the repository's **Security** tab and click **Report a vulnerability**.
* Or contact the maintainers directly via email.

Please include:
1. Description of the vulnerability.
2. Steps to reproduce or a minimal proof-of-concept.
3. Potential impact.

We will acknowledge receipt within 48 hours and work with you to remediate and publish a security advisory.
