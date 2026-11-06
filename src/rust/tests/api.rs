//! Integrationstester som kör mot den riktiga routern.

use std::sync::Arc;

use axum::body::Body;
use axum::http::{Request, StatusCode};
use http_body_util::BodyExt;
use repo_template_api::build_app;
use repo_template_api::config::Config;
use tower::ServiceExt;

fn app() -> axum::Router {
    let config = Config {
        port: 8080,
        version: "test".to_string(),
        api_host: None,
        cors_origins: None,
    };
    build_app(Arc::new(config))
}

async fn get(app: axum::Router, uri: &str) -> (StatusCode, String) {
    let response = app
        .oneshot(
            Request::builder()
                .uri(uri)
                .body(Body::empty())
                .expect("ogiltig request"),
        )
        .await
        .expect("request misslyckades");

    let status = response.status();
    let body = response
        .into_body()
        .collect()
        .await
        .expect("kunde inte läsa body")
        .to_bytes();
    let body = String::from_utf8(body.to_vec()).expect("body är inte utf-8");

    (status, body)
}

#[tokio::test]
async fn health_returns_ok() {
    let (status, body) = get(app(), "/health").await;

    assert_eq!(status, StatusCode::OK);
    assert!(body.contains("\"status\":\"ok\""), "body: {body}");
    assert!(body.contains("\"version\":\"test\""), "body: {body}");
    assert!(body.contains("timestamp"), "body: {body}");
}

#[tokio::test]
async fn hello_returns_greeting() {
    let (status, body) = get(app(), "/hello").await;

    assert_eq!(status, StatusCode::OK);
    assert!(body.contains("Hello, World!"), "body: {body}");
}

#[tokio::test]
async fn openapi_spec_contains_all_routes() {
    let (status, body) = get(app(), "/docs/openapi.json").await;

    assert_eq!(status, StatusCode::OK);
    assert!(body.contains("\"/health\""), "body: {body}");
    assert!(body.contains("\"/hello\""), "body: {body}");
    assert!(body.contains("\"openapi\""), "body: {body}");
    assert!(body.contains("Repo Template Rust API"), "body: {body}");
}

#[tokio::test]
async fn docs_serves_scalar_ui() {
    let (status, body) = get(app(), "/docs").await;

    assert_eq!(status, StatusCode::OK);
    assert!(body.contains("@scalar/api-reference"), "body: {body}");
    assert!(body.contains("/docs/openapi.json"), "body: {body}");
}

#[tokio::test]
async fn unknown_route_returns_404() {
    let (status, _) = get(app(), "/nope").await;

    assert_eq!(status, StatusCode::NOT_FOUND);
}
