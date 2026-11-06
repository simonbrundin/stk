# Neon som dev-databas

Dev-miljön har **ingen Postgres i klustret**. `tilt up` deployar i stället
API:erna mot Neon (Lakebase Postgres, EU-regionen).

## Filer

| Fil                 | Gitignoras | Beskrivning                                   |
| ------------------- | ---------- | ---------------------------------------------- |
| `neon.env`          | Ja         | Anslutningssträngar (skälligheter)             |
| `neon.env.example`  | Nej        | Mall - kopiera till `neon.env`                 |
| `kustomization.yaml`| Nej        | Genererar Secreten `repo-template-db-neon`     |

## Secretens nycklar

| Nyckel                | Endpoint          | Används av                    |
| --------------------- | ----------------- | ----------------------------- |
| `DATABASE_URL`        | poolad (`-pooler`)| Appens databasanslutning      |
| `DATABASE_URL_UNPOOLED` | direkt          | psql, migrationer             |
| `uri`                 | direkt            | Atlas Operator (målsdatabas)  |

`uri` saknar `channel_binding=require` - Atlas använder `lib/pq`, som inte
stöder parametern.

## Steg för att byta databas

1. Skriv `neon.env` (utgångspunkt: `neon.env.example`).
2. Deploya: `tilt up` (eller `kustomize build environments/dev | kubectl apply -f -`).

## Schema

Schemat appliceras av **Atlas Operator** i klustret, mot Neon.
`environments/base/db/schema.sql` är den deklarativa källan - ändra den
filen, inte databasen.

Atlas deployar en egen dev-databas-pod i klustret (`<cr>-atlas-dev-db`)
och diffar schemat mot den. Ingen extra databas behövs på Neon.

## pgAdmin

`../pgadmin/pgadmin-neon-values.yaml` genereras ur `neon.env` av
`./sync-neon-env.sh` och innehåller servernsdefinitionen (med lösenord).
Filen gitignoras.
