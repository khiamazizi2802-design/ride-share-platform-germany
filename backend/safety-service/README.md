# Safety & Verification Service

This microservice handles driver identity verification and secure document processing for the German ride-sharing platform.

## Features

- **Identity Verification**: Integration with POSTIDENT (Mocked) for digital ID verification.
- **P-Schein Validation**: Endpoint for submission and tracking of German passenger transport permits.
- **Secure Document Storage**: All uploaded documents (IDs, criminal records) are encrypted at rest using AES-256-GCM.
- **GDPR Compliance**: Built with privacy-first principles for handling sensitive PII.

## API Endpoints

- `POST /verify/identity`: Initiates POSTIDENT verification case.
- `POST /verify/p-schein`: Submits P-Schein details for manual review.
- `POST /upload-document`: Securely uploads and encrypts driver documentation.

## Tech Stack

- **Language**: Go
- **Encryption**: AES-256-GCM
- **Identity**: POSTIDENT REST API (Mocked)
