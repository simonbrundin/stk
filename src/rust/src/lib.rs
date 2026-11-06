//! Repo Template Rust API.
//!
//! Biblioteket exponerar [`build_app`] så att både binären och
//! integrationstester kan bygga samma router.

pub mod config;
pub mod docs;
pub mod openapi;
pub mod routes;

use std::sync::Arc;

use axum::routing::get;
use axum::{Json, Router};
use tower_http::cors::{Any, CorsLayer};
use tower_http::trace::TraceLayer;
use utoipa_axum::router::OpenApiRouter;
use utoipa_axum::routes;

use crate::config::Config;

/// Bygger hela API:t: endpoints, Scalar UI och OpenAPI-spec.
///
/// OpenApiRouter genererar både axum-routern och OpenAPI-specen ur samma
/// `#[utoipa::path]`-annotationer – specen kan därför inte avvika från koden.
pub fn build_app(config: Arc<Config>) -> Router {
    let state = config.clone();

    let (router, spec) =
        OpenApiRouter::<Arc<Config>>::with_openapi(openapi::openapi(config.api_host.as_deref()))
            .routes(routes!(routes::health::health))
            .routes(routes!(routes::greeting::hello))
            .split_for_parts();

    let spec = Arc::new(spec);

    let app = router
        .route("/docs", get(docs::scalar_ui))
        .route(
            "/docs/openapi.json",
            get(move || {
                let spec = Arc::clone(&spec);
                async move { Json(spec.as_ref().clone()) }
            }),
        )
        .with_state(state)
        .layer(TraceLayer::new_for_http());

    match config.cors_origins.as_ref() {
        Some(origins) => app.layer(cors_layer(origins)),
        None => app,
    }
}

/// Bygger CORS-lager.
///
/// `CORS_ORIGINS=*` → alla origins tillåtna (utveckling).
/// Annars en explicit lista (produktion går via nginx och är same-origin).
fn cors_layer(origins: &[String]) -> CorsLayer {
    if origins.iter().any(|origin| origin == "*") {
        return CorsLayer::new()
            .allow_origin(Any)
            .allow_methods([
                axum::http::Method::GET,
                axum::http::Method::POST,
                axum::http::Method::PUT,
                axum::http::Method::PATCH,
                axum::http::Method::DELETE,
                axum::http::Method::OPTIONS,
            ])
            .allow_headers([
                axum::http::header::ORIGIN,
                axum::http::header::CONTENT_TYPE,
                axum::http::header::ACCEPT,
                axum::http::header::AUTHORIZATION,
            ]);
    }

    let parsed_origins = origins
        .iter()
        .filter_map(|origin| match origin.parse::<axum::http::HeaderValue>() {
            Ok(value) => Some(value),
            Err(error) => {
                tracing::warn!("ogiltig CORS-origin {origin}: {error}");
                None
            }
        })
        .collect::<Vec<_>>();

    CorsLayer::new()
        .allow_origin(parsed_origins)
        .allow_credentials(true)
        .allow_methods([
            axum::http::Method::GET,
            axum::http::Method::POST,
            axum::http::Method::PUT,
            axum::http::Method::PATCH,
            axum::http::Method::DELETE,
            axum::http::Method::OPTIONS,
        ])
        .allow_headers([
            axum::http::header::ORIGIN,
            axum::http::header::CONTENT_TYPE,
            axum::http::header::ACCEPT,
            axum::http::header::AUTHORIZATION,
        ])
}
