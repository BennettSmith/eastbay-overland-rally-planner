# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

This repository implements the backend service. The API contract and behavioral definitions live in the spec repository.
Each service release should declare which spec version it implements (see spec.lock).

Notes:
- Spec repo changelog: what the system does (contract/behavior requirements)
- Service repo changelog: what this deployment changes (runtime/ops/migrations) and which spec version is implemented

## [Unreleased]

### Added

- Updated keycloak config to add ebo-client to ebo realm.
- Added self-service member deletion endpoint (`DELETE /members/me`) with idempotency support.
- Added AWS deployment infrastructure (Terraform + OIDC workflows) with ECS migrations task support.

### Changed
- Added cors support to caddy #17 (AP)
- Member deletion now removes the member’s RSVPs and anonymizes/deactivates the member profile (trips retained).

### Deprecated

### Removed

### Fixed

### Security


