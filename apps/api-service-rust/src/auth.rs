//! JWT verification against the auth-service's remote JWKS
//! ($AUTH_SERVICE_URL/api/auth/jwks). Port of internal/auth/jwks.go: the key
//! set is warmed at startup non-fatally (the server must boot while the auth
//! backend is down), served from cache, and refetched when a token arrives
//! with an unknown kid (key rotation), rate-limited to once a minute.

use std::collections::HashSet;
use std::str::FromStr;
use std::sync::Arc;
use std::time::{Duration, Instant};

use jsonwebtoken::jwk::{AlgorithmParameters, EllipticCurve, Jwk, JwkSet};
use jsonwebtoken::{decode, decode_header, Algorithm, DecodingKey, Validation};
use tokio::sync::{Mutex, RwLock};

#[derive(Debug, Clone)]
pub struct AuthUser {
    pub id: String,
    /// Global role (e.g. "admin") from the "role" claim.
    pub role: Option<String>,
    // Claims below are carried for parity with the Go auth.User (used there
    // only by the unused RequireOrg middleware).
    #[allow(dead_code)]
    pub active_organization_id: Option<String>,
    #[allow(dead_code)]
    pub active_team_id: Option<String>,
    #[allow(dead_code)]
    pub active_org_role: Option<String>,
}

struct CachedKey {
    kid: Option<String>,
    key: DecodingKey,
    alg: Algorithm,
}

pub struct JwksVerifier {
    jwks_url: String,
    http: reqwest::Client,
    keys: RwLock<Vec<CachedKey>>,
    last_refresh: Mutex<Option<Instant>>,
}

impl JwksVerifier {
    /// Registers the JWKS endpoint and warms it. A fetch failure is logged but
    /// non-fatal, matching the Go auth.Init behaviour.
    pub async fn new(auth_service_url: &str) -> Arc<Self> {
        let v = Arc::new(Self {
            jwks_url: format!("{auth_service_url}/api/auth/jwks"),
            http: reqwest::Client::new(),
            keys: RwLock::new(Vec::new()),
            last_refresh: Mutex::new(None),
        });
        if let Err(err) = v.refresh().await {
            tracing::warn!("auth: failed to warm JWKS {} (will retry lazily): {err}", v.jwks_url);
        }
        // Periodic refresh so rotated-out keys eventually drop from the cache
        // even without an unknown-kid trigger.
        let bg = Arc::clone(&v);
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(Duration::from_secs(15 * 60));
            interval.tick().await; // skip the immediate first tick
            loop {
                interval.tick().await;
                if let Err(err) = bg.refresh().await {
                    tracing::warn!("auth: JWKS refresh: {err}");
                }
            }
        });
        v
    }

    async fn refresh(&self) -> Result<(), String> {
        let set: JwkSet = self
            .http
            .get(&self.jwks_url)
            .send()
            .await
            .map_err(|e| format!("fetch jwks {}: {e}", self.jwks_url))?
            .error_for_status()
            .map_err(|e| format!("fetch jwks {}: {e}", self.jwks_url))?
            .json()
            .await
            .map_err(|e| format!("decode jwks {}: {e}", self.jwks_url))?;

        let mut keys = Vec::new();
        for jwk in &set.keys {
            match (DecodingKey::from_jwk(jwk), jwk_algorithm(jwk)) {
                (Ok(key), Some(alg)) => keys.push(CachedKey { kid: jwk.common.key_id.clone(), key, alg }),
                _ => tracing::warn!("auth: skipping unusable JWK (kid={:?})", jwk.common.key_id),
            }
        }
        *self.last_refresh.lock().await = Some(Instant::now());
        *self.keys.write().await = keys;
        Ok(())
    }

    /// Refetches the JWKS if it hasn't been refreshed in the last minute.
    /// Called on unknown-kid tokens so key rotation is picked up promptly
    /// without letting bad tokens hammer the auth service.
    async fn refresh_rate_limited(&self) {
        {
            let last = self.last_refresh.lock().await;
            if matches!(*last, Some(t) if t.elapsed() < Duration::from_secs(60)) {
                return;
            }
        }
        if let Err(err) = self.refresh().await {
            tracing::warn!("auth: JWKS refresh: {err}");
        }
    }

    pub async fn verify_token(&self, token: &str) -> Result<AuthUser, String> {
        let header = decode_header(token).map_err(|e| format!("verify token: {e}"))?;

        if let Some(user) = self.try_keys(token, header.kid.as_deref()).await? {
            return Ok(user);
        }
        // No cached key matched the token's kid — the set may have rotated.
        self.refresh_rate_limited().await;
        match self.try_keys(token, header.kid.as_deref()).await? {
            Some(user) => Ok(user),
            None => Err("verify token: no matching key".to_string()),
        }
    }

    /// Tries the token against cached keys matching its kid (all keys when the
    /// token has no kid, as jwx does). Ok(None) means no candidate key was
    /// found; Err means a candidate existed but verification failed.
    async fn try_keys(&self, token: &str, kid: Option<&str>) -> Result<Option<AuthUser>, String> {
        let keys = self.keys.read().await;
        let mut last_err = None;
        let mut tried = false;
        for cached in keys.iter() {
            if let Some(kid) = kid {
                if cached.kid.as_deref() != Some(kid) {
                    continue;
                }
            }
            tried = true;
            let mut validation = Validation::new(cached.alg);
            // Go verifies exp/nbf when present but requires neither, and
            // checks no audience/issuer (jwt.WithValidate(true)).
            validation.required_spec_claims = HashSet::new();
            validation.validate_exp = true;
            validation.validate_nbf = true;
            validation.validate_aud = false;
            match decode::<serde_json::Value>(token, &cached.key, &validation) {
                Ok(data) => return user_from_claims(&data.claims).map(Some),
                Err(e) => last_err = Some(format!("verify token: {e}")),
            }
        }
        if tried {
            Err(last_err.unwrap_or_else(|| "verify token: verification failed".to_string()))
        } else {
            Ok(None)
        }
    }
}

/// Resolves the signing algorithm for a JWK: its declared "alg" when present,
/// otherwise inferred from the key type.
fn jwk_algorithm(jwk: &Jwk) -> Option<Algorithm> {
    if let Some(alg) = jwk.common.key_algorithm {
        if let Ok(alg) = Algorithm::from_str(&alg.to_string()) {
            return Some(alg);
        }
    }
    match &jwk.algorithm {
        AlgorithmParameters::RSA(_) => Some(Algorithm::RS256),
        AlgorithmParameters::EllipticCurve(p) => match p.curve {
            EllipticCurve::P256 => Some(Algorithm::ES256),
            EllipticCurve::P384 => Some(Algorithm::ES384),
            _ => None,
        },
        AlgorithmParameters::OctetKeyPair(_) => Some(Algorithm::EdDSA),
        _ => None,
    }
}

fn user_from_claims(claims: &serde_json::Value) -> Result<AuthUser, String> {
    let mut id = string_claim(claims, "id");
    if id.is_empty() {
        id = string_claim(claims, "sub");
    }
    if id.is_empty() {
        return Err("missing id claim".to_string());
    }
    Ok(AuthUser {
        id,
        role: opt_string_claim(claims, "role"),
        active_organization_id: opt_string_claim(claims, "activeOrganizationId"),
        active_team_id: opt_string_claim(claims, "activeTeamId"),
        active_org_role: opt_string_claim(claims, "activeOrgRole"),
    })
}

fn string_claim(claims: &serde_json::Value, key: &str) -> String {
    claims.get(key).and_then(|v| v.as_str()).unwrap_or("").to_string()
}

fn opt_string_claim(claims: &serde_json::Value, key: &str) -> Option<String> {
    match claims.get(key).and_then(|v| v.as_str()) {
        Some(v) if !v.is_empty() => Some(v.to_string()),
        _ => None,
    }
}

pub fn parse_bearer(auth_header: &str) -> Option<&str> {
    auth_header.strip_prefix("Bearer ")
}
