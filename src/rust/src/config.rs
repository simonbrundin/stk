use std::env;

/// Konfiguration som läses från miljövariabler.
///
/// Paritet med Go-API:t i `src/go`:
/// - `PORT`          – port att lyssna på (default 8080, prefixet `:` accepteras)
/// - `APP_VERSION`   – version som visas i `/health`
/// - `API_HOST`      – värd som sätts i OpenAPI-specens `servers`
/// - `CORS_ORIGINS`  – komma-separerad lista, `*` = alla origins
#[derive(Debug, Clone)]
pub struct Config {
    pub port: u16,
    pub version: String,
    pub api_host: Option<String>,
    pub cors_origins: Option<Vec<String>>,
}

impl Config {
    pub fn from_env() -> Self {
        let port = env::var("PORT")
            .ok()
            .map(|value| value.trim_start_matches(':').trim().to_string())
            .and_then(|value| value.parse::<u16>().ok())
            .unwrap_or(8080);

        let version = env::var("APP_VERSION").unwrap_or_else(|_| "dev".to_string());

        let api_host = env::var("API_HOST")
            .ok()
            .map(|value| value.trim().to_string())
            .filter(|value| !value.is_empty());

        let cors_origins = env::var("CORS_ORIGINS")
            .ok()
            .map(|value| {
                value
                    .split(',')
                    .map(|origin| origin.trim().to_string())
                    .filter(|origin| !origin.is_empty())
                    .collect::<Vec<_>>()
            })
            .filter(|origins| !origins.is_empty());

        Self {
            port,
            version,
            api_host,
            cors_origins,
        }
    }
}

impl Default for Config {
    fn default() -> Self {
        Self::from_env()
    }
}
