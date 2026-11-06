//! Startpunkt för API:et.
//!
//! ```text
//! cargo run                # startar API:t
//! cargo run -- healthcheck # TCP-probe mot den egna porten (används i Docker HEALTHCHECK)
//! ```

use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Duration;

use repo_template_api::build_app;
use repo_template_api::config::Config;
use tracing::info;
use tracing_subscriber::EnvFilter;

#[tokio::main]
async fn main() {
    init_tracing();

    let config = Arc::new(Config::from_env());
    let port = config.port;

    // `api healthcheck` används av Dockerfile:s HEALTHCHECK (distroless saknar
    // curl/wget, så vi använder samma binär för att göra en TCP-probe).
    if std::env::args().nth(1).as_deref() == Some("healthcheck") {
        std::process::exit(healthcheck(port));
    }

    let app = build_app(config);

    let address = SocketAddr::from(([0, 0, 0, 0], port));
    let listener = match tokio::net::TcpListener::bind(address).await {
        Ok(listener) => listener,
        Err(error) => {
            tracing::error!("kunde inte binda {address}: {error}");
            std::process::exit(1);
        }
    };

    info!("Repo Template Rust API lyssnar på {address}");
    info!("Scalar UI: http://localhost:{port}/docs");

    if let Err(error) = axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await
    {
        tracing::error!("serverfel: {error}");
        std::process::exit(1);
    }
}

/// TCP-probe mot den egna API-porten. Returnerar exit-kod 0/1.
fn healthcheck(port: u16) -> i32 {
    let address = format!("127.0.0.1:{port}");
    match std::net::TcpStream::connect_timeout(
        &address.parse().expect("ogiltig adress"),
        Duration::from_secs(2),
    ) {
        Ok(_) => 0,
        Err(error) => {
            eprintln!("healthcheck misslyckades ({address}): {error}");
            1
        }
    }
}

fn init_tracing() {
    let filter = EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info"));
    tracing_subscriber::fmt()
        .with_env_filter(filter)
        .with_target(false)
        .init();
}

/// Graceful shutdown: SIGINT (Ctrl+C) och SIGTERM (Kubernetes).
async fn shutdown_signal() {
    let ctrl_c = async {
        let _ = tokio::signal::ctrl_c().await;
    };

    #[cfg(unix)]
    let terminate = async {
        match tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate()) {
            Ok(mut signal) => {
                signal.recv().await;
            }
            Err(error) => {
                tracing::warn!("kunde inte lyssna på SIGTERM: {error}");
                std::future::pending::<()>().await;
            }
        }
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {}
        _ = terminate => {}
    }

    info!("shutting down");
}
