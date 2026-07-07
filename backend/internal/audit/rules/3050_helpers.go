// Package rules — Sprint 33 Fase 3 helpers.
//
// Helpers compartilhados por S09/S13 (DiasUteis, ÚltimoDiaUtilMes) e
// validações de dataBase em geral.
package rules

import "time"

// Feriados nacionais fixos (lei federal — não mudam ano a ano).
// Formato: "MM-DD".
var feriadosNacionaisFixos = map[string]bool{
	"01-01": true, // Confraternização Universal
	"04-21": true, // Tiradentes
	"05-01": true, // Dia do Trabalho
	"09-07": true, // Independência
	"10-12": true, // Nossa Senhora Aparecida
	"11-02": true, // Finados
	"11-15": true, // Proclamação da República
	"12-25": true, // Natal
}

// pascoa retorna a data do Domingo de Páscoa para um dado ano (algoritmo
// de Gauss/Computus). Usado para calcular feriados móveis.
//
// Referência: https://en.wikipedia.org/wiki/Date_of_Easter#Anonymous_Gregorian_algorithm
func pascoa(ano int) time.Time {
	a := ano % 19
	b := ano / 100
	c := ano % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	mes := (h + l - 7*m + 114) / 31
	dia := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(ano, time.Month(mes), dia, 0, 0, 0, 0, time.UTC)
}

// feriadosMoveis retorna mapa de feriados móveis para um ano (Carnaval,
// Sexta-Feira Santa, Corpus Christi). Baseado na data da Páscoa.
//
//   - Carnaval: 47 dias antes da Páscoa (terça-feira)
//   - Sexta-Feira Santa: 2 dias antes da Páscoa
//   - Corpus Christi: 60 dias depois da Páscoa
func feriadosMoveis(ano int) map[string]bool {
	p := pascoa(ano)
	carnaval := p.AddDate(0, 0, -47)
	sextaSanta := p.AddDate(0, 0, -2)
	corpusChristi := p.AddDate(0, 0, 60)
	return map[string]bool{
		carnaval.Format("01-02"):      true, // terça de Carnaval
		sextaSanta.Format("01-02"):    true,
		corpusChristi.Format("01-02"): true,
	}
}

// IsDiaUtilBACEN retorna true se data for dia útil no calendário
// bancário BACEN (exclui sábado, domingo e feriados nacionais).
//
// Placeholder consciente: feriados estaduais/municipais não são
// considerados. Evolução futura = API BACEN ou tabela anual atualizável.
//
// Sábado = 6, Domingo = 7 em time.Weekday().
func IsDiaUtilBACEN(data time.Time) bool {
	w := data.Weekday()
	if w == time.Saturday || w == time.Sunday {
		return false
	}
	key := data.Format("01-02")
	if feriadosNacionaisFixos[key] {
		return false
	}
	if moveis := feriadosMoveis(data.Year()); moveis[key] {
		return false
	}
	return true
}

// IsUltimoDiaUtilMes retorna true se data for o último dia útil do mês
// no calendário BACEN.
//
// Comportamento atual:
//   - data == último dia do mês E data é dia útil → true
//   - data == último dia do mês E data NÃO é útil (sábado/domingo/feriado) → false
//   - data != último dia do mês → false
//
// Edge case conhecido: se o último dia do mês cai em sábado (ex: 2025-05-31),
// o "último dia útil" BACEN real seria a sexta anterior. Esta implementação
// retorna false nesse caso — semântica não-bacen. Carry-over para Fase 4.
//
// Placeholder consciente: a SCD raramente erra nesse caso (próximo a zero).
// Evolução futura = checagem que volta dias até encontrar último dia útil.
func IsUltimoDiaUtilMes(data time.Time) bool {
	ano, mes, _ := data.Date()
	// Último dia do mês: dia 1 do mês seguinte - 1 dia
	primeiroProximo := time.Date(ano, mes+1, 1, 0, 0, 0, 0, time.UTC)
	ultimo := primeiroProximo.AddDate(0, 0, -1)
	if !data.Equal(ultimo) {
		return false
	}
	return IsDiaUtilBACEN(data)
}
