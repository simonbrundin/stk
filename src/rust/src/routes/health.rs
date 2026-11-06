use axum::Json;
use axum::extract::State;
use chrono::Utc;
use serde::Serialize;
use utoipa::ToSchema;

use crate::config::Config;

/// Svar från `/health`.
#[derive(Debug, Serialize, ToSchema)]
pub struct HealthResponse {
    /// Status för tjänsten.
    #[schema(example = "ok")]
    pub status: &'static str,
    /// Version av den körande tjänsten (`APP_VERSION`).
    #[schema(example = "1.0.0")]
    pub version: String,
    /// Tidsstämpel i ISO 8601 (UTC).
    #[schema(example = "2024-01-15T10:30:00Z")]
    pub timestamp: String,
}

/// Hälsokontroll.
///
/// Returnerar tjänstens status, version och aktuell tid.
#[utoipa::path(
    get,
    path = "/health",
    tag = "health",
    responses(
        (status = 200, description = "Tjänsten är frisk", body = HealthResponse)
    )
)]
pub async fn health(State(config): State<std::sync::Arc<Config>>) -> Json<HealthResponse> {
    Json(HealthResponse {
        status: "ok",
        version: config.version.clone(),
        timestamp: Utc::now().to_rfc3339(),
    })
}
