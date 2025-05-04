<!-- ~/.GH/Qompass/Go/Azimuth/discord/docs/architecture.md -->
<!-- ----------------------------------------------------- -->
<!-- Copyright (C) 2025 Qompass AI, All rights reserved -->

 ┌──────────┐   WebSocket     ┌──────── ─┐  HTTPS/MTLS  ┌───────────┐
 │ Discord  │◀───────────────▶│  Bot     │────────────▶ │  rose LLM │
 │  Client  │                 │(Go + SDK)│◀──────────── ┤           │
 └──────────┘                 └────┬─────┘              └─────┬─────┘
                                   │                       ┌─▼────────┐
         IMAP/SMTP   ┌──────────┐  │   NATS JetStream      │ Postgres │
   SMS Webhook (TLS) │ Email/SMS│──┼──────────────────────▶│  DB      │
 ───────────────────▶│ Gateway  │  │       events          └──────────┘
                     └──────────┘  │
                                   ▼
                         ┌────────────────┐
                         │  Observability │ Prometheus + Grafana + Loki
                         └────────────────┘

