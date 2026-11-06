use axum::Json;
use serde::Serialize;
use utoipa::ToSchema;

/// Svar från `/hello`.
#[derive(Debug, Serialize, ToSchema)]
pub struct HelloResponse {
    /// Hälsning.
    #[schema(example = "Hello, World!")]
    pub message: String,
}

/// Enkel exempel-endpoint.
#[utoipa::path(
    get,
    path = "/hello",
    tag = "greeting",
    responses(
        (status = 200, description = "Hälsning", body = HelloResponse)
    )
)]
pub async fn hello() -> Json<HelloResponse> {
    Json(HelloResponse {
        message: "Hello, World!".to_string(),
    })
}
