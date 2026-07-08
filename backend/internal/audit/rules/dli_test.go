// Tests para CADOC 2062 — DLI (Demonstrativo de Limites Operacionais).
package rules

import (
	"context"
	"strings"
	"testing"
)

func TestDLI01CNPJValido(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		cnpj    string
		wantErr bool
	}{
		{"valido_8_digitos", "12345678", false},
		{"vazio", "", true},
		{"menos_de_8", "1234567", true},
		{"mais_de_8", "123456789", true},
		{"com_letras", "1234567A", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDLI{Root: DocDLIRoot{CNPJ: tt.cnpj}}
			err := DLI01CNPJValido{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("CNPJ=%q: err=%v, wantErr=%v", tt.cnpj, err, tt.wantErr)
			}
		})
	}
}

func TestDLI02DataBaseValido(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		db      string
		wantErr bool
	}{
		{"valido_2024_03", "2024-03", false},
		{"valido_2024_12", "2024-12", false},
		{"valido_jan", "2024-01", false},
		{"invalido_mes_00", "2024-00", true},
		{"invalido_mes_13", "2024-13", true},
		{"invalido_formato", "2024/03", true},
		{"invalido_curto", "2024-3", true},
		{"vazio", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDLI{Root: DocDLIRoot{DataBase: tt.db}}
			err := DLI02DataBaseValido{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("DataBase=%q: err=%v, wantErr=%v", tt.db, err, tt.wantErr)
			}
		})
	}
}

func TestDLI03TipoEnvioValido(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		tipo    string
		wantErr bool
	}{
		{"inclusao_I", "I", false},
		{"substituicao_S", "S", false},
		{"vazio", "", true},
		{"outro_valor", "X", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDLI{TipoEnvio: tt.tipo}
			err := DLI03TipoEnvioValido{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("tipoEnvio=%q: err=%v, wantErr=%v", tt.tipo, err, tt.wantErr)
			}
		})
	}
}

func TestDLI04CodigoDocumentoValido(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		cod     string
		wantErr bool
	}{
		{"valido_2062", "2062", false},
		{"invalido_outro", "3040", true},
		{"vazio", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDLI{Root: DocDLIRoot{CodigoDoc: tt.cod}}
			err := DLI04CodigoDocumentoValido{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("codigoDoc=%q: err=%v, wantErr=%v", tt.cod, err, tt.wantErr)
			}
		})
	}
}

func TestDLI05TemConteudo(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		limites    []DLILimite
		parametros []DLIParametro
		accounts   map[string]float64
		wantErr    bool
	}{
		{"com_limites", []DLILimite{{Codigo: "06.00"}}, nil, nil, false},
		{"com_contas", nil, nil, map[string]float64{"6.10.01": 1000}, false},
		{"vazio_total", nil, nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDLI{Limites: tt.limites, Parametros: tt.parametros, Accounts: tt.accounts}
			if doc.Accounts == nil {
				doc.Accounts = make(map[string]float64)
			}
			err := DLI05TemConteudo{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestDLI06LimiteCodigoValido(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		limites []DLILimite
		wantErr bool
	}{
		{"valido_NN_NN", []DLILimite{{Codigo: "06.00"}}, false},
		{"valido_20_00", []DLILimite{{Codigo: "20.00"}}, false},
		{"invalido_sem_ponto", []DLILimite{{Codigo: "0600"}}, true},
		{"invalido_tres_segmentos", []DLILimite{{Codigo: "06.00.00"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDLI{Limites: tt.limites}
			err := DLI06LimiteCodigoValido{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestDLI08ContaCOSIFValida(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		accounts map[string]float64
		wantErr  bool
	}{
		{"valido_3_segmentos", map[string]float64{"6.10.01": 1000}, false},
		{"valido_4_segmentos", map[string]float64{"6.90.05.01": 500}, false},
		{"invalido_formato", map[string]float64{"invalid": 100}, true},
		{"negativo", map[string]float64{"6.10.01": -100}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDLI{Accounts: tt.accounts}
			err := DLI08ContaCOSIFValida{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestDLI09PLAContabil(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		accounts map[string]float64
		wantErr  bool
	}{
		{"pla_valido", map[string]float64{"6.10.01": 1000, "6.10.02": 100, "6.10.90": 50}, false},
		{"6_10_01_ausente", map[string]float64{"6.10.02": 100, "6.10.90": 50}, true},
		{"pla_negativo", map[string]float64{"6.10.01": 100, "6.10.02": -100, "6.10.90": 150}, true}, // PLA = -150 <= 0
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDLI{Accounts: tt.accounts}
			err := DLI09PLAContabil{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestDLI10MargemPLA(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		accounts map[string]float64
		wantErr  bool
	}{
		{"margem_consistente", map[string]float64{"6.00.00": 500, "6.10.01": 1000, "6.10.02": 100, "6.10.90": 50, "6.90.00": 550}, false},
		{"margem_inconsistente", map[string]float64{"6.00.00": 0, "6.10.01": 1000, "6.10.02": 100, "6.10.90": 50, "6.90.00": 550}, false}, // 0 = not reported
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDLI{Accounts: tt.accounts}
			err := DLI10MargemPLA{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestDLI13LimitePartesRelacionadas(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		accounts map[string]float64
		limites  []DLILimite
		wantErr  bool
	}{
		{"dentro_limite", map[string]float64{"6.10.01": 1000, "6.10.02": 0, "6.10.90": 0}, []DLILimite{{Codigo: "20.00", Valor: "199"}}, false},
		{"excede_20pct", map[string]float64{"6.10.01": 1000, "6.10.02": 0, "6.10.90": 0}, []DLILimite{{Codigo: "20.00", Valor: "201"}}, true},
		{"sem_limite", map[string]float64{"6.10.01": 1000, "6.10.02": 0, "6.10.90": 0}, nil, false}, // limite opcional
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDLI{Accounts: tt.accounts, Limites: tt.limites}
			err := DLI13LimitePartesRelacionadas{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestParseDocDLI(t *testing.T) {
	xml := `<documentoDLI cnpj="12345678" dataBase="2024-03" codigoDocumento="2062" tipoEnvio="I">
		<limitesInformados>
			<limite codigoLimite="06.00" enviado="S">1000000.00</limite>
		</limitesInformados>
		<parametros>
			<parametro codigoParametro="31" valorParametro="João Silva"/>
		</parametros>
		<contas>
			<conta codigoConta="6.10.01" valorConta="5000000.00"/>
		</contas>
	</documentoDLI>`

	doc, err := ParseDocDLI([]byte(xml))
	if err != nil {
		t.Fatalf("ParseDocDLI failed: %v", err)
	}

	if doc.Root.CNPJ != "12345678" {
		t.Errorf("CNPJ=%q, want 12345678", doc.Root.CNPJ)
	}
	if doc.Root.DataBase != "2024-03" {
		t.Errorf("DataBase=%q, want 2024-03", doc.Root.DataBase)
	}
	if len(doc.Limites) != 1 {
		t.Fatalf("len(Limites)=%d, want 1", len(doc.Limites))
	}
	if doc.Limites[0].Codigo != "06.00" {
		t.Errorf("Limite[0].Codigo=%q, want 06.00", doc.Limites[0].Codigo)
	}
	if doc.Accounts["6.10.01"] != 5000000.00 {
		t.Errorf("Accounts[6.10.01]=%v, want 5000000.00", doc.Accounts["6.10.01"])
	}
}

func TestSomaPLA(t *testing.T) {
	accounts := map[string]float64{
		"6.10.01": 1000,
		"6.10.02": 100,
		"6.10.90": 50,
	}
	pla := SomaPLA(accounts)
	// 1000 + 100 - 50 = 1050
	if pla != 1050 {
		t.Errorf("SomaPLA=%v, want 1050", pla)
	}
}

func TestSomaCapitalRealizado(t *testing.T) {
	accounts := map[string]float64{"8.10.01": 500}
	if SomaCapitalRealizado(accounts) != 500 {
		t.Errorf("SomaCapitalRealizado=%v, want 500", SomaCapitalRealizado(accounts))
	}
}

func TestXDDLI01CNPJConsistente(t *testing.T) {
	ctx := context.Background()
	// Setup parsedDLI and parsedDRL with same CNPJ
	parsedDLI = &DocDLI{Root: DocDLIRoot{CNPJ: "12345678"}}
	parsedDRL = &DocDRL{Root: DocDRLRoot{CNPJ: "12345678"}}

	// Should pass — CNPJs match
	err := XDDLI01CNPJConsistente{}.Apply(ctx, nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Change DRL CNPJ — should fail
	parsedDRL.Root.CNPJ = "87654321"
	err = XDDLI01CNPJConsistente{}.Apply(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "CNPJ DLI") {
		t.Errorf("expected CNPJ mismatch error, got %v", err)
	}

	// Clean up
	parsedDLI = nil
	parsedDRL = nil
}

func TestXDDLI06NSFRxLCRConsistente(t *testing.T) {
	ctx := context.Background()
	// Setup: NSFR >= 100% and LCR < 80% — should alert
	parsedDLP = &DocDLP{NSFRRatio: 110}
	parsedDRL = &DocDRL{LCRRatio: 70}

	err := XDDLI06NSFRxLCRConsistente{}.Apply(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "NSFR") {
		t.Errorf("expected NSFR/LCR inconsistency error, got %v", err)
	}

	// Clean up
	parsedDLP = nil
	parsedDRL = nil
}
