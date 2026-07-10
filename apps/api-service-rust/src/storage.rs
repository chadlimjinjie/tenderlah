//! Uploads files (e.g. onboarding logos) to a Supabase Storage bucket via its
//! plain HTTP API (no SDK needed). Port of internal/storage/supabase.go.

pub struct SupabaseStorage {
    base_url: String,
    service_key: String,
    bucket: String,
    http: reqwest::Client,
}

impl SupabaseStorage {
    pub fn new(base_url: String, service_key: String, bucket: String) -> Self {
        Self { base_url, service_key, bucket, http: reqwest::Client::new() }
    }

    /// POSTs data at path within the bucket and returns its public URL.
    /// Caller picks path (e.g. userId/logo/uuid.ext) to avoid collisions.
    pub async fn upload(&self, path: &str, content_type: &str, data: Vec<u8>) -> Result<String, String> {
        let url = format!("{}/storage/v1/object/{}/{}", self.base_url, self.bucket, path);
        let resp = self
            .http
            .post(&url)
            .bearer_auth(&self.service_key)
            .header(reqwest::header::CONTENT_TYPE, content_type)
            .body(data)
            .send()
            .await
            .map_err(|e| e.to_string())?;

        let status = resp.status();
        if status.as_u16() >= 300 {
            let body = resp.text().await.unwrap_or_default();
            return Err(format!("supabase upload failed: {status}: {body}"));
        }
        Ok(format!("{}/storage/v1/object/public/{}/{}", self.base_url, self.bucket, path))
    }
}
