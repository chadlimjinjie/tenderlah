//! OpenAI-compatible chat client for Ollama cloud, ported from
//! internal/llm/ollama.go. Non-streaming; optional Bearer key; optional
//! `response_format: json_object`.

use serde::{Deserialize, Serialize};

pub const DEFAULT_MODEL: &str = "gpt-oss:120b-cloud";

pub struct LlmClient {
    pub base_url: String,
    pub api_key: String,
    pub model: String,
    http: reqwest::Client,
}

#[derive(Serialize)]
struct Message<'a> {
    role: &'a str,
    content: &'a str,
}

#[derive(Serialize)]
struct ResponseFormat {
    #[serde(rename = "type")]
    kind: &'static str,
}

#[derive(Serialize)]
struct ChatRequest<'a> {
    model: &'a str,
    messages: [Message<'a>; 2],
    #[serde(skip_serializing_if = "Option::is_none")]
    response_format: Option<ResponseFormat>,
    stream: bool,
}

#[derive(Deserialize)]
struct ChatResponse {
    #[serde(default)]
    choices: Vec<Choice>,
}

#[derive(Deserialize)]
struct Choice {
    message: ChoiceMessage,
}

#[derive(Deserialize)]
struct ChoiceMessage {
    #[serde(default)]
    content: String,
}

impl LlmClient {
    pub fn new(base_url: String, api_key: String, model: String) -> Self {
        Self { base_url, api_key, model, http: reqwest::Client::new() }
    }

    /// Sends a chat completion and returns the assistant message content.
    #[allow(dead_code)]
    pub async fn generate(&self, system: &str, prompt: &str) -> Result<String, String> {
        self.generate_inner(system, prompt, false).await
    }

    /// Requests a JSON-formatted response.
    pub async fn generate_json(&self, system: &str, prompt: &str) -> Result<String, String> {
        self.generate_inner(system, prompt, true).await
    }

    async fn generate_inner(&self, system: &str, prompt: &str, json_mode: bool) -> Result<String, String> {
        let req = ChatRequest {
            model: &self.model,
            messages: [
                Message { role: "system", content: system },
                Message { role: "user", content: prompt },
            ],
            response_format: json_mode.then_some(ResponseFormat { kind: "json_object" }),
            stream: false,
        };

        let mut http_req = self
            .http
            .post(format!("{}/v1/chat/completions", self.base_url))
            .json(&req);
        if !self.api_key.is_empty() {
            http_req = http_req.bearer_auth(&self.api_key);
        }

        let resp = http_req.send().await.map_err(|e| format!("ollama request: {e}"))?;
        let status = resp.status();
        if status.as_u16() >= 400 {
            let body = resp.text().await.unwrap_or_default();
            return Err(format!("ollama {}: {}", status.as_u16(), body));
        }

        let result: ChatResponse = resp.json().await.map_err(|e| format!("decode response: {e}"))?;
        match result.choices.into_iter().next() {
            Some(choice) => Ok(choice.message.content),
            None => Err("no choices in response".to_string()),
        }
    }
}
