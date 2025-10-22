# --- Estágio 1: Builder ---
# Usamos uma imagem oficial do Go (baseada em Alpine para ser mais leve)
# Dê um nome a este estágio, como "builder", para que possamos nos referir a ele mais tarde.
FROM golang:1.24-alpine AS builder

# Define o diretório de trabalho dentro do contêiner
WORKDIR /app

# 1. Copie os arquivos de módulo e baixe as dependências PRIMEIRO
# Isso aproveita o cache de camadas do Docker. Se 'go.mod' e 'go.sum' não mudarem,
# o Docker não precisará baixar as dependências novamente.
# (Assumindo que você tem um 'go.mod' junto com o 'go.sum' que enviou)
COPY go.mod go.sum ./
RUN go mod download

# 2. Copie todo o código-fonte do seu projeto
COPY . .

# 3. Compile a aplicação
# - CGO_ENABLED=0: Desabilita o CGO para criar um binário estático.
# - GOOS=linux: Garante que o build seja para Linux (o SO do contêiner).
# - -ldflags '-w -s': Remove informações de debug e símbolos,
#   reduzindo drasticamente o tamanho do binário final.
# - -o /go-release-manager: Define o nome e local do binário de saída.
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -s' -o /go-release-manager ./main.go

# --- Estágio 2: Final ---
# Começamos com uma imagem Alpine nova e minúscula.
# É uma das menores imagens Linux disponíveis (cerca de 5MB).
FROM alpine:latest

# IMPORTANTE: Seu código (em 'create.go' e 'internal/git') parece
# executar comandos 'git' para criar tags, fazer push, etc.
# Por causa disso, precisamos instalar o 'git' na imagem final.
# Se seu código NÃO dependesse do 'git' CLI, poderíamos usar a imagem 'scratch'
# que é 100% vazia e ainda menor.
RUN apk --no-cache add git

# Define o diretório de trabalho
WORKDIR /app

# Copia APENAS o binário compilado do estágio "builder"
COPY --from=builder /go-release-manager /go-release-manager

# Define o ponto de entrada. Quando o contêiner for executado,
# ele irá executar este binário.
ENTRYPOINT ["/go-release-manager"]

# Define um comando padrão (opcional, mas uma boa prática)
# Se o usuário não passar nenhum argumento (como "create"),
# o contêiner executará "go-release-manager --help".
CMD []