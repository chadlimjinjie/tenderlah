/// Returns the env var's value, or `fallback` when unset/empty. Exits the
/// process when the var is unset and there is no fallback (required vars),
/// matching the Go service's mustEnv.
pub fn must_env(key: &str, fallback: &str) -> String {
    match std::env::var(key) {
        Ok(v) if !v.is_empty() => v,
        _ => {
            if fallback.is_empty() {
                eprintln!("required env var {key} not set");
                std::process::exit(1);
            }
            fallback.to_string()
        }
    }
}
