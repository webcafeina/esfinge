# Esfinge — compilación y comprobaciones.
#
# Go no está en el PATH del sistema: se instaló en ~/.local/go para no tocar
# nada fuera del $HOME.
GO   ?= $(HOME)/.local/go/bin/go
PNPM ?= pnpm

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
DIST    := dist
FRONT   := frontend

.PHONY: todo
todo: comprobar esfinge

## comprobar: vet, los tests de Go y los tipos de la interfaz
.PHONY: comprobar
comprobar:
	$(GO) vet ./...
	$(GO) vet -tags dev ./...
	$(GO) test ./...
	cd $(FRONT) && $(PNPM) exec tsc -b --noEmit

## contraste: mide las parejas de color de los dos temas y las lista
.PHONY: contraste
contraste:
	$(GO) test ./internal/tema/ -run TestContrasteDeLosDosTemas -v

## tokens: regenera los colores y medidas que consume la interfaz
.PHONY: tokens
tokens:
	$(GO) run ./cmd/tokens

## icono: rasteriza los SVG a los PNG que piden los sistemas y la portada
.PHONY: icono
icono:
	cd $(FRONT) && $(PNPM) run icono

## descargas: reescribe la tabla de descargas del README para una versión
.PHONY: descargas
descargas:
	@herramientas/actualizar-descargas.sh "$(VERSION)"

## ventana-dmg: dibuja cómo quedará la ventana del DMG, sin necesidad de un Mac
.PHONY: ventana-dmg
ventana-dmg:
	cd $(FRONT) && node herramientas/ventana-dmg.mjs

## dmg: la imagen de disco de macOS. Solo funciona en un Mac con create-dmg
.PHONY: dmg
dmg: app
	@empaquetado/macos/armar-dmg.sh "$(VERSION)"

## e2e: mueve la interfaz de verdad contra el Go de verdad, en los dos temas
.PHONY: e2e
e2e:
	cd $(FRONT) && $(PNPM) exec playwright test

## frontend: construye la interfaz y la deja donde la aplicación la embebe
.PHONY: frontend
frontend: tokens
	cd $(FRONT) && $(PNPM) install --frozen-lockfile && $(PNPM) run build
	@rm -rf internal/interfaz/dist && mkdir -p internal/interfaz/dist
	@cp -r $(FRONT)/dist/. internal/interfaz/dist/

## app: la aplicación con ventana, para este sistema
##
## Necesita la herramienta wails y, en Linux, webkit2gtk y pkg-config. En la
## máquina de desarrollo no están, así que esto se ejecuta en integración
## continua o en el Mac. La parte de Go sí compila en cualquier sitio.
.PHONY: app
app: frontend
	wails build -ldflags "$(LDFLAGS)"

## esfinge: la línea de comandos, para este sistema
.PHONY: esfinge
esfinge:
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o esfinge ./cmd/esfinge

## puente: el proceso que lanza el navegador para hablar con la bóveda
##
## En Windows va con «-H windowsgui», que **no le quita la entrada y la salida
## estándar** —los descriptores los pasa quien lanza el proceso— y sí le quita la
## ventana de consola que si no parpadearía cada pocos minutos mientras se navega.
.PHONY: puente
puente:
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o esfinge-puente ./cmd/esfinge-puente

## extension: construye la extensión del navegador, para Chrome y para Firefox
##
## Salen dos porque **los manifiestos no son el mismo**: Chrome quiere un
## service_worker y Firefox una lista de scripts, y el identificador lo elige uno
## en Firefox y lo asigna la tienda en Chrome.
.PHONY: extension
extension:
	cd navegador && $(PNPM) install --frozen-lockfile && $(PNPM) run build
	cd navegador && NAVEGADOR=firefox $(PNPM) exec vite build
	@ls -la navegador/dist/*/

## dev: levanta el Go de verdad para poder mover la interfaz en el navegador
.PHONY: dev
dev:
	@echo "Interfaz en http://127.0.0.1:5173 — en otra terminal: cd frontend && pnpm dev"
	$(GO) run -tags dev ./cmd/dev

PLATAFORMAS := \
	linux/amd64 linux/arm64 \
	darwin/amd64 darwin/arm64 \
	windows/amd64 windows/arm64

## publicar: comprueba y compila la línea de comandos para los seis objetivos
.PHONY: publicar
publicar: comprobar publicar-cli

## publicar-cli: los seis binarios de la línea de comandos, sin comprobar antes
##
## Sin cgo no hay nada que enlazar del sistema, y por eso estos sí cruzan de
## plataforma desde aquí. La aplicación con ventana no puede: necesita el webview
## de cada sistema. Va separado de «publicar» para que la publicación automática
## no repita las comprobaciones, que ya corren en su propio flujo.
.PHONY: publicar-cli
publicar-cli:
	@mkdir -p $(DIST)
	@rm -f $(DIST)/esfinge-*-linux-* $(DIST)/esfinge-*-darwin-* \
	       $(DIST)/esfinge-*-windows-* $(DIST)/SHA256SUMS
	@for p in $(PLATAFORMAS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		nombre="esfinge-$(VERSION)-$$os-$$arch$$ext"; \
		echo "  $$nombre"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" \
			-o "$(DIST)/$$nombre" ./cmd/esfinge || exit 1; \
		puente="esfinge-puente-$(VERSION)-$$os-$$arch$$ext"; \
		echo "  $$puente"; \
		enlazador="$(LDFLAGS)"; \
		[ "$$os" = "windows" ] && enlazador="$$enlazador -H windowsgui"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$$enlazador" \
			-o "$(DIST)/$$puente" ./cmd/esfinge-puente || exit 1; \
	done
	@cd $(DIST) && sha256sum esfinge-* > SHA256SUMS
	@ls -lh $(DIST)

## instalar: compila e instala la línea de comandos en esta máquina Linux
.PHONY: instalar
instalar: esfinge
	@chmod +x empaquetado/linux/instalar.sh
	@cp esfinge empaquetado/linux/esfinge
	@empaquetado/linux/instalar.sh; estado=$$?; rm -f empaquetado/linux/esfinge; exit $$estado

## limpiar: borra lo construido
.PHONY: limpiar
limpiar:
	rm -rf $(DIST) esfinge $(FRONT)/dist internal/interfaz/dist/assets
	rm -rf $(FRONT)/test-results $(FRONT)/playwright-report

## ayuda: esta lista
.PHONY: ayuda
ayuda:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
