package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Config é a estrutura principal do arquivo .go-releaserc.yml
type Config struct {
	ReleaseRules []ReleaseRule `yaml:"releaseRules"`
}

// ReleaseRule define como um tipo de commit afeta a versão
type ReleaseRule struct {
	Type    string `yaml:"type"`
	Release string `yaml:"release"` // "major", "minor", "patch", "none"
}

// defaultConfig retorna a configuração padrão (o comportamento atual)
// caso nenhum .go-releaserc.yml seja encontrado.
func defaultConfig() *Config {
	return &Config{
		ReleaseRules: []ReleaseRule{
			{Type: "feat", Release: "minor"},
			{Type: "fix", Release: "patch"},
			// Por padrão, outros tipos não geram release
			{Type: "docs", Release: "none"},
			{Type: "style", Release: "none"},
			{Type: "refactor", Release: "none"},
			{Type: "perf", Release: "none"},
			{Type: "test", Release: "none"},
			{Type: "chore", Release: "none"},
			{Type: "build", Release: "none"},
			{Type: "ci", Release: "none"},
		},
	}
}

// LoadConfig procura, lê e analisa o arquivo .go-releaserc.yml.
// Se não encontrar, retorna a configuração padrão.
func LoadConfig() (*Config, error) {
	configFileName := ".go-releaserc.yml"

	// 1. Tenta ler o arquivo de configuração
	data, err := os.ReadFile(configFileName)
	if err != nil {
		// Se o erro for 'file not found', não é um erro fatal.
		// Apenas usamos a configuração padrão.
		if os.IsNotExist(err) {
			log.Println("Nenhum .go-releaserc.yml encontrado. Usando regras padrão (feat/fix).")
			return defaultConfig(), nil
		}
		// Outro erro (ex: permissão de leitura)
		return nil, err
	}

	// 2. Arquivo encontrado, vamos analisá-lo (parse)
	log.Println("Arquivo .go-releaserc.yml encontrado. Carregando regras personalizadas.")

	// Começa com os padrões, para que o usuário precise definir apenas o que quer mudar
	config := defaultConfig()

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
