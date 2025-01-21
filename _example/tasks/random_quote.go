package tasks

import (
	"context"

	"math/rand"
)

func RandomQuote(ctx context.Context) (string, error) {
	quotes := []string{
		"O caminho da felicidade ainda existe. É uma trilha estreita em meio à selva triste!",
		"Onde estiver, seja lá como for, tenha fé porque até no lixão nasce flor.",
		"A alma guarda o que a mente tenta esquecer!",
		"A vida é louca e nela estou de passagem",
	}

	return quotes[rand.Intn(len(quotes))], nil
}
