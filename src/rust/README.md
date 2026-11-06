# Rust API (`src/rust`)

REST-API för repo-template byggt med det mest etablerade Rust-stacket för
REST-APIs: **Axum + Tokio + Tower + utoipa (OpenAPI) + Scalar UI**.

Paritet med Go-API:t i [`src/go`](../go): samma endpoints, samma
miljövariabler och samma dokumentationsupplägg (`/docs` + `/docs/openapi.json`).

## Stack och varför

| Del             | Crate / verktyg         | Skäl                                                                 |
| --------------- | ----------------------- | -------------------------------------------------------------------- |
| HTTP-server     | **Axum** 0.8            | De facto-standard för Rust-webben; bygger på Tokio/hyper/Tower.        |
| Runtime         | **Tokio**               | Asynkron I/O, signaler och graceful shutdown.                         |
| Mellanlager     | **tower-http**          | CORS, tracing och andra standardlager.                                |
| Serialisering   | **serde** + **serde_json** | Standard i Rust-ekosystemet.                                          |
| OpenAPI         | **utoipa** 6 (+ **utoipa-axum**) | Code-first: specen genereras ur `#[utoipa::path]` och kan inte avvika från koden. |
| Dokumentation   | **Scalar UI**           | Samma UI som Go-API:t (mörkt tema), läser OpenAPI-specen.             |
| Loggning        | **tracing**             | Strukturerad loggning, filtreras med `RUST_LOG`.                      |

## Struktur

```
src/rust/
├── Cargo.toml
├── Cargo.lock
├── rust-toolchain.toml      # pinnar toolchain (1.99.1) + clippy/rustfmt
├── Dockerfile               # multi-stage: musl-static → distroless
├── src/
│   ├── main.rs              # entrypoint: bind, graceful shutdown, healthcheck
│   ├── lib.rs               # build_app() – samma router används av testerna
│   ├── config.rs            # env-var-konfiguration
│   ├── openapi.rs           # OpenAPI-metadata (title, version, tags, servers)
│   ├── docs.rs              # Scalar UI (HTML)
│   └── routes/
│       ├── health.rs        # GET /health
│       └── greeting.rs      # GET /hello
└── tests/
    └── api.rs               # integrationstester mot den riktiga routern
```

## Kommandon

```bash
cd src/rust

cargo run                # starta API:t (PORT=8080)
cargo test               # integrationstester
cargo clippy --all-targets -- -D warnings
cargo fmt                # formatering
cargo build --release    # release-build

# Docker
docker build -t repo-template-api-rust .
docker run --rm -p 8080:8080 repo-template-api-rust
```

## Endpoints

| Endpoint            | Metod | Beskrivning                                        |
| ------------------- | ----- | --------------------------------------------------- |
| `/health`           | GET   | Status, version och tidsstämpel                     |
| `/hello`            | GET   | Enkel hälsning                                       |
| `/docs`             | GET   | Scalar UI (Swagger/OpenAPI-referens)                |
| `/docs/openapi.json`| GET   | OpenAPI 3.1-specen (genereras ur koden)              |

## Miljövariabler

| Variabel       | Default | Beskrivning                                                        |
| -------------- | ------- | ------------------------------------------------------------------- |
| `PORT`         | `8080`  | Port att lyssna på (`:8080` accepteras också)                       |
| `APP_VERSION`  | `dev`   | Version som visas i `/health`                                        |
| `API_HOST`     | -       | Sätts i OpenAPI-specens `servers` (t.ex. `https://api.example.com`) |
| `CORS_ORIGINS` | -       | Kommaseparerad lista eller `*`. Utan variabel är CORS avstängd.      |
| `RUST_LOG`     | `info`  | Loggnivå för `tracing` (`debug`, `info`, `warn`, `error`)           |

## Lägga till en endpoint

1. Skapa/modifiera en modul i `src/routes/` och annotera handlern:

   ```rust
   #[utoipa::path(get, path = "/users", tag = "users",
       responses((status = 200, body = Vec<User>)))]
   pub async fn list_users() -> Json<Vec<User>> { /* ... */ }
   ```

2. Registrera routen i `src/lib.rs` – **en `.routes(routes!(...))` per path**:

   ```rust
   .routes(routes!(routes::users::list_users))
   ```

3. Kör `cargo test` – OpenAPI-specen uppdateras automatiskt, ingen
   genererad fil behöver committeras.

> **Obs:** `routes!(a, b)` lägger båda handlers i samma method-router och
> orsakar "Overlapping method route" om de har olika paths – använd en
> `.routes()`-rad per endpoint.

## Skillnader mot Go-API:t

- **Ingen genererad OpenAPI-fil.** Go använder swaggo + `swagger.json` på disk.
  Rust-versionen är code-first: `utoipa` bygger specen vid runtime, vilket gör
  att specen aldrig kan bli inaktuell.
- **Scalar laddas från CDN** (`cdn.jsdelivr.net`) istället för att serveras
  från en inbäddad bundle. Ska Scalar fungera utan internet kan HTML:en byggas
  om med `self-hosted` / `standalone`-varianten från Scalar.
- **Ingen databaskoppling ännu**, i linje med Go-API:t just nu.

## Kubernetes / Tilt

API:t är kopplat till klustret:

| Fil                                       | Syfte                                                     |
| ----------------------------------------- | --------------------------------------------------------- |
| `environments/base/rust-api-deployment.yaml` | Deployment (image `rust-api:latest`, httpGet-probe mot `/health`) |
| `environments/base/rust-api-service.yaml`   | ClusterIP-service `rust-api-service` på port 8080         |
| `environments/base/kustomization.yaml`      | Registrerar deployment + service i base-overlayn         |
| `environments/dev/Tiltfile`                 | `docker_build("rust-api", "../../src/rust")` + port-forward + länkar till `/health` och `/docs` |

```bash
cd environments/dev && tilt up          # bygger och deployar rust-api:t
```

Tilt bygger imagen från `src/rust/Dockerfile` och sätter image-referensen i
Deploymenten automatiskt (`rust-api:latest` → `localhost:5000/rust-api:tilt-*`).
Base-overlayn är refererad av `environments/dev`, `environments/pr` och
`environments/prod`, så deploymenten finns i alla miljöer.

**Utanför scope just nu:**

- `environments/prod` och `environments/pr` har ingen image-override – där pekar
  alla images på `:latest` (gäller Go-API:t lika). Detta är en repo-wide lucka,
  inte specifikt för Rust.
- Ingen databaskoppling (t.ex. `sqlx`) ännu.
