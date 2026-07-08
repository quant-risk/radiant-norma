// Tests para DRM leiaute (CADOC 2060).
package rules

import (
	"context"
	"strings"
	"testing"
)

func TestDRM01HeaderValido(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		root    DocDRMLeiauteRoot
		wantErr bool
	}{
		{"ok", DocDRMLeiauteRoot{IdDocto: "2060", DataBase: "2024-03", IdInstFinanc: "12345678", TipoArq: "I"}, false},
		{"iddocto_errado", DocDRMLeiauteRoot{IdDocto: "3040", DataBase: "2024-03", IdInstFinanc: "12345678", TipoArq: "I"}, true},
		{"database_vazio", DocDRMLeiauteRoot{IdDocto: "2060", DataBase: "", IdInstFinanc: "12345678", TipoArq: "I"}, true},
		{"cnpj_vazio", DocDRMLeiauteRoot{IdDocto: "2060", DataBase: "2024-03", IdInstFinanc: "", TipoArq: "I"}, true},
		{"tipoarq_invalido", DocDRMLeiauteRoot{IdDocto: "2060", DataBase: "2024-03", IdInstFinanc: "12345678", TipoArq: "X"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDRMLeiaute{Root: tt.root}
			err := DRM01HeaderValido{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestDRM02ItensObrigatorios(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		ativos      []ItemCarteiraLeiaute
		passivos    []ItemCarteiraLeiaute
		derivativos []ItemCarteiraLeiaute
		atf         []ItemCarteiraLeiaute
		wantErr     bool
	}{
		{"com_ativos", []ItemCarteiraLeiaute{{Item: "001"}}, nil, nil, nil, false},
		{"vazio", nil, nil, nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDRMLeiaute{
				Ativos:                tt.ativos,
				Passivos:              tt.passivos,
				Derivativos:           tt.derivativos,
				AtividadesFinanceiras: tt.atf,
			}
			err := DRM02ItensObrigatorios{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestDRM04ValorMaMRequerido(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		ativos  []ItemCarteiraLeiaute
		wantErr bool
	}{
		{"vertice_12_com_mam", []ItemCarteiraLeiaute{{Item: "001", Fluxos: []FluxoVerticeLeiaute{{CodVertice: "12", ValorMaM: 100}}}}, false},
		{"vertice_12_sem_mam", []ItemCarteiraLeiaute{{Item: "001", Fluxos: []FluxoVerticeLeiaute{{CodVertice: "12", ValorMaM: 0}}}}, true},
		{"vertice_11_sem_mam", []ItemCarteiraLeiaute{{Item: "001", Fluxos: []FluxoVerticeLeiaute{{CodVertice: "11", ValorMaM: 0}}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDRMLeiaute{Ativos: tt.ativos}
			err := DRM04ValorMaMRequerido{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestDRM05ValorAlocadoPositivo(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		ativos  []ItemCarteiraLeiaute
		wantErr bool
	}{
		{"positivo", []ItemCarteiraLeiaute{{Item: "001", Fluxos: []FluxoVerticeLeiaute{{ValorAlocado: 100}}}}, false},
		{"zero", []ItemCarteiraLeiaute{{Item: "001", Fluxos: []FluxoVerticeLeiaute{{ValorAlocado: 0}}}}, false},
		{"negativo", []ItemCarteiraLeiaute{{Item: "001", Fluxos: []FluxoVerticeLeiaute{{ValorAlocado: -100}}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDRMLeiaute{Ativos: tt.ativos}
			err := DRM05ValorAlocadoPositivo{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestDRM07FatorRiscoValido(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		ativos  []ItemCarteiraLeiaute
		wantErr bool
	}{
		{"jm1_valido", []ItemCarteiraLeiaute{{Item: "001", FatorRisco: "JM1"}}, false},
		{"jt2_valido", []ItemCarteiraLeiaute{{Item: "001", FatorRisco: "JT2"}}, false},
		{"ji9_valido", []ItemCarteiraLeiaute{{Item: "001", FatorRisco: "JI9"}}, false},
		{"ff1_valido", []ItemCarteiraLeiaute{{Item: "001", FatorRisco: "FF1"}}, false},
		{"invalido", []ItemCarteiraLeiaute{{Item: "001", FatorRisco: "XXX"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &DocDRMLeiaute{Ativos: tt.ativos}
			err := DRM07FatorRiscoValido{}.Apply(ctx, doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestParseDocDRMLeiaute(t *testing.T) {
	xml := `<DRM>
		<IdDocto>2060</IdDocto>
		<IdDoctoVersao>v01</IdDoctoVersao>
		<DataBase>2024-03</DataBase>
		<IdInstFinanc>12345678</IdInstFinanc>
		<TipoArq>I</TipoArq>
		<NomeContato>João Silva</NomeContato>
		<FoneContato>1199999999</FoneContato>
		<Ativo>
			<ItemCarteira Item="001" FatorRisco="JM1" LocalRegistro="01" CarteiraNegoc="01">
				<FluxoVertice CodVertice="01" ValorAlocado="1000.00" ValorMaM="950.00"/>
			</ItemCarteira>
		</Ativo>
	</DRM>`

	doc, err := ParseDocDRMLeiaute([]byte(xml))
	if err != nil {
		t.Fatalf("ParseDocDRMLeiaute failed: %v", err)
	}

	if doc.Root.IdDocto != "2060" {
		t.Errorf("IdDocto=%q, want 2060", doc.Root.IdDocto)
	}
	if doc.Root.DataBase != "2024-03" {
		t.Errorf("DataBase=%q, want 2024-03", doc.Root.DataBase)
	}
	if doc.Root.TipoArq != "I" {
		t.Errorf("TipoArq=%q, want I", doc.Root.TipoArq)
	}
	if len(doc.Ativos) != 1 {
		t.Fatalf("len(Ativos)=%d, want 1", len(doc.Ativos))
	}
	if doc.Ativos[0].Item != "001" {
		t.Errorf("Ativos[0].Item=%q, want 001", doc.Ativos[0].Item)
	}
	if doc.Ativos[0].FatorRisco != "JM1" {
		t.Errorf("Ativos[0].FatorRisco=%q, want JM1", doc.Ativos[0].FatorRisco)
	}
	if len(doc.Ativos[0].Fluxos) != 1 {
		t.Fatalf("len(Fluxos)=%d, want 1", len(doc.Ativos[0].Fluxos))
	}
	if doc.Ativos[0].Fluxos[0].ValorAlocado != 1000.0 {
		t.Errorf("ValorAlocado=%v, want 1000", doc.Ativos[0].Fluxos[0].ValorAlocado)
	}
}

func TestParseDocDRMLeiauteDerivativos(t *testing.T) {
	xml := `<DRM>
		<IdDocto>2060</IdDocto>
		<DataBase>2024-03</DataBase>
		<IdInstFinanc>12345678</IdInstFinanc>
		<TipoArq>I</TipoArq>
		<Derivativo>
			<ItemCarteira Item="001" IdPosicao="C" FatorRisco="JT1" LocalRegistro="01" CarteiraNegoc="01">
				<FluxoVertice CodVertice="03" ValorAlocado="500.00" ValorMaM="490.00"/>
			</ItemCarteira>
		</Derivativo>
	</DRM>`

	doc, err := ParseDocDRMLeiaute([]byte(xml))
	if err != nil {
		t.Fatalf("ParseDocDRMLeiaute failed: %v", err)
	}

	if len(doc.Derivativos) != 1 {
		t.Fatalf("len(Derivativos)=%d, want 1", len(doc.Derivativos))
	}
	if doc.Derivativos[0].IdPosicao != "C" {
		t.Errorf("IdPosicao=%q, want C", doc.Derivativos[0].IdPosicao)
	}
}

func TestParseDocDRMLeiauteAtividadeFinanceira(t *testing.T) {
	xml := `<DRM>
		<IdDocto>2060</IdDocto>
		<DataBase>2024-03</DataBase>
		<IdInstFinanc>12345678</IdInstFinanc>
		<TipoArq>I</TipoArq>
		<AtividadeFinanceira>
			<ItemCarteira Item="AEC" IdPosicao="C" FatorRisco="JM1">
				<FluxoVertice CodVertice="05" ValorAlocado="200.00"/>
			</ItemCarteira>
		</AtividadeFinanceira>
	</DRM>`

	doc, err := ParseDocDRMLeiaute([]byte(xml))
	if err != nil {
		t.Fatalf("ParseDocDRMLeiaute failed: %v", err)
	}

	if len(doc.AtividadesFinanceiras) != 1 {
		t.Fatalf("len(AtividadesFinanceiras)=%d, want 1", len(doc.AtividadesFinanceiras))
	}
	if doc.AtividadesFinanceiras[0].Item != "AEC" {
		t.Errorf("Item=%q, want AEC", doc.AtividadesFinanceiras[0].Item)
	}
	if doc.AtividadesFinanceiras[0].Fluxos[0].ValorMaM != 0 {
		t.Errorf("ValorMaM=%v, want 0 for AtividadeFinanceira", doc.AtividadesFinanceiras[0].Fluxos[0].ValorMaM)
	}
}

func TestDRM06AtividadeFinanceiraSemMaM(t *testing.T) {
	ctx := context.Background()
	// AtividadeFinanceira com ValorMaM != 0 should warn
	doc := &DocDRMLeiaute{
		AtividadesFinanceiras: []ItemCarteiraLeiaute{
			{Item: "AEC", Fluxos: []FluxoVerticeLeiaute{{ValorAlocado: 100, ValorMaM: 50}}},
		},
	}
	err := DRM06AtividadeFinanceiraSemMaM{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "não deve ter ValorMaM") {
		t.Errorf("expected AtividadeFinanceira ValorMaM error, got %v", err)
	}

	// ValorMaM = 0 is ok
	doc2 := &DocDRMLeiaute{
		AtividadesFinanceiras: []ItemCarteiraLeiaute{
			{Item: "AEC", Fluxos: []FluxoVerticeLeiaute{{ValorAlocado: 100, ValorMaM: 0}}},
		},
	}
	err = DRM06AtividadeFinanceiraSemMaM{}.Apply(ctx, doc2)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDocDRMLeiauteComputeAggregates(t *testing.T) {
	doc := &DocDRMLeiaute{
		Ativos: []ItemCarteiraLeiaute{
			{Item: "001", Fluxos: []FluxoVerticeLeiaute{{ValorAlocado: 100, ValorMaM: 90}}},
			{Item: "002", Fluxos: []FluxoVerticeLeiaute{{ValorAlocado: 200, ValorMaM: 180}}},
		},
		Passivos: []ItemCarteiraLeiaute{
			{Item: "003", Fluxos: []FluxoVerticeLeiaute{{ValorAlocado: 50, ValorMaM: 40}}},
		},
		Derivativos: []ItemCarteiraLeiaute{
			{Item: "004", Fluxos: []FluxoVerticeLeiaute{{ValorAlocado: 75, ValorMaM: 70}}},
		},
		AtividadesFinanceiras: []ItemCarteiraLeiaute{
			{Item: "AEC", Fluxos: []FluxoVerticeLeiaute{{ValorAlocado: 25}}},
		},
	}
	doc.computeAggregates()

	// 100+200+50+75+25 = 450
	if doc.TotalValorAlocado != 450 {
		t.Errorf("TotalValorAlocado=%v, want 450", doc.TotalValorAlocado)
	}
	// 90+180+40+70+0 = 380 (AtividadeFinanceira não tem ValorMaM)
	if doc.TotalValorMaM != 380 {
		t.Errorf("TotalValorMaM=%v, want 380", doc.TotalValorMaM)
	}
}
