//! Client for the internal embed-service (potion-base-8M, 256-dim).
//!
//! This is independently written to mirror apps/tender-worker/internal/store/embed.go
//! (and the Go api-service's internal/db/embed.go). The services are separate
//! modules, and this is a small, dependency-light function, so duplicating it
//! beats introducing a shared module. Any change here must be mirrored there
//! too — all outputs share the same vector space as the tender.embedding
//! column, and a drift between the copies would silently corrupt relevance
//! ranking.

use serde::{Deserialize, Serialize};

#[derive(Serialize)]
struct EmbedRequest<'a> {
    texts: [&'a str; 1],
}

#[derive(Deserialize)]
struct EmbedResponse {
    vectors: Vec<Vec<f32>>,
}

/// Calls the embed-service to turn text into a 256-dim vector. The service has
/// no auth (internal only).
pub async fn embed(base_url: &str, text: &str) -> Result<Vec<f32>, String> {
    let resp = reqwest::Client::new()
        .post(format!("{base_url}/api/embed"))
        .json(&EmbedRequest { texts: [text] })
        .send()
        .await
        .map_err(|e| e.to_string())?;

    let status = resp.status();
    if status.as_u16() >= 400 {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("embed-service {}: {}", status.as_u16(), body));
    }

    let out: EmbedResponse = resp.json().await.map_err(|e| e.to_string())?;
    out.vectors
        .into_iter()
        .next()
        .ok_or_else(|| "embed-service: empty response".to_string())
}
