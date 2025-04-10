#!/bin/bash

# ----------------------------------------------------------------------------
# Script: build-mac.sh
# Descrição: Automatiza o build de um app Go+Fyne para macOS com microfone.
# Autor: Seu Nome
# Versão: 1.1
# Uso: ./build-mac.sh
# ----------------------------------------------------------------------------

# Configurações
DEV_ID=br.app.seven.delfos
APP_NAME="Delfos"
BUILD_DIR="build"
OUTPUT_APP="$BUILD_DIR/$APP_NAME.app"
DMG_NAME="$BUILD_DIR/$APP_NAME.dmg"

# ----------------------------------------------------------------------------
# Função para verificar dependências
check_dependencies() {
    echo "Verificando dependências..."
    if ! command -v go &> /dev/null; then
        echo "❌ Go não instalado. Instale primeiro: https://golang.org/dl/"
        exit 1
    fi

    if ! command -v fyne &> /dev/null; then
        echo "❌ Fyne CLI não instalado. Instalando..."
        go install fyne.io/fyne/v2/cmd/fyne@latest || exit 1
    fi

    if ! xcode-select -p &> /dev/null; then
        echo "❌ Xcode não instalado. Instale via App Store ou com: xcode-select --install"
        exit 1
    fi
}

# ----------------------------------------------------------------------------
# Função para limpar builds anteriores
clean_previous_builds() {
    echo "Limpando builds anteriores..."
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"
}

# ----------------------------------------------------------------------------
# Função para compilar o aplicativo
build_app() {
    echo "Compilando o aplicativo..."
    fyne build -o "$BUILD_DIR/$APP_NAME" || exit 1

    echo "Empacotando como .app..."
    fyne package -os darwin -icon icon.png -exe "$BUILD_DIR/$APP_NAME" --release || exit 1

    # Move o .app para a pasta build, se necessário
    if [ -d "$APP_NAME.app" ]; then
        echo "Movendo o .app para a pasta build..."
        mv "$APP_NAME.app" "$OUTPUT_APP"
    fi

    # Adiciona permissão de microfone ao Info.plist
    echo "Adicionando permissão de microfone..."
    PLIST_FILE="$OUTPUT_APP/Contents/Info.plist"
    if [ -f "$PLIST_FILE" ]; then
        /usr/libexec/PlistBuddy -c "Add NSMicrophoneUsageDescription string 'Este app precisa acessar o microfone para funcionalidades de áudio.'" "$PLIST_FILE" || echo "⚠️ Não foi possível modificar o Info.plist"
    else
        echo "⚠️ Info.plist não encontrado em $OUTPUT_APP"
    fi
}

# ----------------------------------------------------------------------------
# Função para assinar o aplicativo (opcional)
sign_app() {
    if [ -z "$DEV_ID" ]; then
        echo "⚠️ Certificado de desenvolvedor não configurado. Pule a assinatura ou defina DEV_ID."
        return
    fi

    echo "Assinando o aplicativo..."
    codesign --deep --force --options=runtime --sign "$DEV_ID" "$OUTPUT_APP" || echo "⚠️ Falha na assinatura"
}

# ----------------------------------------------------------------------------
# Função para criar DMG (opcional)
create_dmg() {
    echo "Criando DMG..."
    hdiutil create -volname "$APP_NAME" -srcfolder "$OUTPUT_APP" -ov -format UDZO "$DMG_NAME" || echo "⚠️ Falha ao criar DMG"
}

# ----------------------------------------------------------------------------
# Execução principal
echo "🛠️  Iniciando build de $APP_NAME para macOS..."

check_dependencies
clean_previous_builds
build_app

# Descomente as linhas abaixo se necessário:
# sign_app
# create_dmg

echo "✅ Build concluído! App disponível em: $OUTPUT_APP"
