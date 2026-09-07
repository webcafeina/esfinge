# Esfinge — compilación y comprobaciones.
#
# Go no está en el PATH del sistema: se instaló en ~/.local/go para no tocar
# nada fuera del $HOME.
GO ?= $(HOME)/.local/go/bin/go

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
DIST    := dist

# Sin cgo no hay nada que enlazar del sistema, y por eso los binarios cruzan de
# plataforma sin un compilador de C por medio.
export CGO_ENABLED = 0

PLATAFORMAS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

.PHONY: todo
todo: comprobar esfinge

## esfinge: compila el binario para esta máquina
.PHONY: esfinge
esfinge:
	$(GO) build -ldflags "$(LDFLAGS)" -o esfinge ./cmd/esfinge

## comprobar: vet y toda la batería de pruebas, contraste incluido
.PHONY: comprobar
comprobar:
	$(GO) vet ./...
	$(GO) test ./...

## contraste: mide las parejas de color de los dos temas y las lista
.PHONY: contraste
contraste:
	$(GO) test ./internal/ui/ -run TestContrasteDeLosDosTemas -v

## publicar: los seis binarios en dist/
.PHONY: publicar
publicar: comprobar
	@# Se borran los binarios sueltos, no el directorio entero: si se vaciara,
	@# «make linux» detrás de «make macos» se llevaría por delante el ZIP que
	@# acaba de crearse. Para dejarlo todo limpio está «make limpiar».
	@mkdir -p $(DIST)
	@rm -f $(DIST)/esfinge-*-linux-* $(DIST)/esfinge-*-darwin-* \
	       $(DIST)/esfinge-*-windows-* $(DIST)/SHA256SUMS
	@for p in $(PLATAFORMAS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		nombre="esfinge-$(VERSION)-$$os-$$arch$$ext"; \
		echo "  $$nombre"; \
		GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" \
			-o "$(DIST)/$$nombre" ./cmd/esfinge || exit 1; \
	done
	@cd $(DIST) && sha256sum * > SHA256SUMS
	@echo
	@ls -lh $(DIST)

## paquetes: los dos paquetes con instalador, macOS y Linux
.PHONY: paquetes
paquetes: macos linux

## instalar: compila e instala en esta máquina Linux
.PHONY: instalar
instalar: esfinge
	@chmod +x empaquetado/linux/instalar.sh
	@cp esfinge empaquetado/linux/esfinge
	@empaquetado/linux/instalar.sh; estado=$$?; rm -f empaquetado/linux/esfinge; exit $$estado

## linux: el tar.gz con instalador para Linux
.PHONY: linux
linux: publicar
	@rm -rf $(DIST)/linux && mkdir -p "$(DIST)/linux/esfinge-$(VERSION)"
	@cp "$(DIST)/esfinge-$(VERSION)-linux-amd64" "$(DIST)/linux/esfinge-$(VERSION)/esfinge-linux-amd64"
	@cp "$(DIST)/esfinge-$(VERSION)-linux-arm64" "$(DIST)/linux/esfinge-$(VERSION)/esfinge-linux-arm64"
	@cp empaquetado/linux/instalar.sh "$(DIST)/linux/esfinge-$(VERSION)/"
	@chmod +x "$(DIST)/linux/esfinge-$(VERSION)/"*
	@tar -C $(DIST)/linux -czf "$(DIST)/esfinge-$(VERSION)-linux.tar.gz" "esfinge-$(VERSION)"
	@rm -rf $(DIST)/linux
	@ls -lh "$(DIST)/esfinge-$(VERSION)-linux.tar.gz"

## macos: el ZIP con instalador de doble clic para el cliente
.PHONY: macos
macos: publicar
	@rm -rf $(DIST)/macos && mkdir -p "$(DIST)/macos/Esfinge $(VERSION)"
	@cp "$(DIST)/esfinge-$(VERSION)-darwin-arm64" "$(DIST)/macos/Esfinge $(VERSION)/esfinge-apple-silicon"
	@cp "$(DIST)/esfinge-$(VERSION)-darwin-amd64" "$(DIST)/macos/Esfinge $(VERSION)/esfinge-intel"
	@cp empaquetado/macos/* "$(DIST)/macos/Esfinge $(VERSION)/"
	@chmod +x "$(DIST)/macos/Esfinge $(VERSION)/Instalar Esfinge.command" \
	          "$(DIST)/macos/Esfinge $(VERSION)/esfinge-apple-silicon" \
	          "$(DIST)/macos/Esfinge $(VERSION)/esfinge-intel"
	@cd $(DIST)/macos && zip -r -q "../Esfinge-$(VERSION)-macOS.zip" "Esfinge $(VERSION)"
	@rm -rf $(DIST)/macos
	@ls -lh $(DIST)/Esfinge-$(VERSION)-macOS.zip

## limpiar: borra los binarios
.PHONY: limpiar
limpiar:
	rm -rf $(DIST) esfinge

## ayuda: esta lista
.PHONY: ayuda
ayuda:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
