use utoipa::OpenApi;
use utoipa::openapi::server::Server;

/// Grund för OpenAPI-specen.
///
/// Specen är *code-first*: den genereras ur `#[utoipa::path]`-annotationerna på
/// handlers, vilket garanterar att dokumentationen alltid matchar implementationen.
#[derive(OpenApi)]
#[openapi(
    info(
        title = "Repo Template Rust API",
        version = "1.0.0",
        description = "REST-API byggd med Axum och Tokio. OpenAPI-specen (Swagger 3.1) genereras \
                       ur koden med utoipa och serveras tillsammans med Scalar UI.",
        license(name = "MIT")
    ),
    tags(
        (name = "health", description = "Hälsokontroll"),
        (name = "greeting", description = "Exempel-endpoints"),
        (name = "docs", description = "API-dokumentation")
    )
)]
pub struct ApiDoc;

/// Bygger OpenAPI-specen.
///
/// `api_host` sätts i `servers` när den är satt – samma semantik som Go-API:t
/// (`API_HOST` → `host` i Swagger-specen).
pub fn openapi(api_host: Option<&str>) -> utoipa::openapi::OpenApi {
    let mut spec = <ApiDoc as utoipa::OpenApi>::openapi();

    if let Some(host) = api_host.map(str::trim).filter(|host| !host.is_empty()) {
        let url = if host.starts_with("http://") || host.starts_with("https://") {
            host.to_string()
        } else {
            format!("http://{host}")
        };
        spec.servers = Some(vec![Server::new(url)]);
    }

    spec
}
