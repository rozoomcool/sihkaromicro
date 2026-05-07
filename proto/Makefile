.PHONY: gen lint breaking tidy

# ── Generate ──────────────────────────────────────────
gen:
	buf generate

# ── Lint ─────────────────────────────────────────────
lint:
	buf lint

# ── Breaking changes ──────────────────────────────────
breaking:
	buf breaking --against '.git#branch=main'

# ── Tidy ─────────────────────────────────────────────
tidy:
	go mod tidy