package domain

import (
	"testing"
	"time"
)

func TestAppelOffreZeroValue(t *testing.T) {
	var ao AppelOffre
	if ao.ID != "" {
		t.Errorf("expected empty ID, got %q", ao.ID)
	}
	if ao.Titre != "" {
		t.Errorf("expected empty Titre, got %q", ao.Titre)
	}
	if ao.MaitreDouvrage != "" {
		t.Errorf("expected empty MaitreDouvrage, got %q", ao.MaitreDouvrage)
	}
	if !ao.DateLimite.IsZero() {
		t.Errorf("expected zero DateLimite, got %v", ao.DateLimite)
	}
	if ao.CCTP != "" {
		t.Errorf("expected empty CCTP, got %q", ao.CCTP)
	}
	if ao.CCAP != "" {
		t.Errorf("expected empty CCAP, got %q", ao.CCAP)
	}
	if ao.BPU_DPGF != "" {
		t.Errorf("expected empty BPU_DPGF, got %q", ao.BPU_DPGF)
	}
}

func TestAppelOffreFieldAssignment(t *testing.T) {
	deadline := time.Date(2024, 12, 31, 17, 0, 0, 0, time.UTC)
	ao := AppelOffre{
		ID:                     "AO-001",
		Titre:                  "Construction d'un bâtiment hospitalier",
		MaitreDouvrage:         "CHU de Lyon",
		DateLimite:             deadline,
		RegelementConsultation: "RC-2024",
		CCTP:                   "Clauses techniques spéciales",
		CCAP:                   "Clauses administratives spéciales",
		BPU_DPGF:               "Bordereau des prix unitaires",
	}

	if ao.ID != "AO-001" {
		t.Errorf("expected ID AO-001, got %q", ao.ID)
	}
	if ao.Titre != "Construction d'un bâtiment hospitalier" {
		t.Errorf("unexpected Titre: %q", ao.Titre)
	}
	if ao.MaitreDouvrage != "CHU de Lyon" {
		t.Errorf("unexpected MaitreDouvrage: %q", ao.MaitreDouvrage)
	}
	if !ao.DateLimite.Equal(deadline) {
		t.Errorf("expected deadline %v, got %v", deadline, ao.DateLimite)
	}
	if ao.RegelementConsultation != "RC-2024" {
		t.Errorf("unexpected RegelementConsultation: %q", ao.RegelementConsultation)
	}
	if ao.CCTP != "Clauses techniques spéciales" {
		t.Errorf("unexpected CCTP: %q", ao.CCTP)
	}
	if ao.CCAP != "Clauses administratives spéciales" {
		t.Errorf("unexpected CCAP: %q", ao.CCAP)
	}
	if ao.BPU_DPGF != "Bordereau des prix unitaires" {
		t.Errorf("unexpected BPU_DPGF: %q", ao.BPU_DPGF)
	}
}

func TestReponseHistoriqueZeroValue(t *testing.T) {
	var r ReponseHistorique
	if r.ID != "" {
		t.Errorf("expected empty ID, got %q", r.ID)
	}
	if r.AppelOffreID != "" {
		t.Errorf("expected empty AppelOffreID, got %q", r.AppelOffreID)
	}
	if r.ExigenceTechnique != "" {
		t.Errorf("expected empty ExigenceTechnique, got %q", r.ExigenceTechnique)
	}
	if r.ReponseApportee != "" {
		t.Errorf("expected empty ReponseApportee, got %q", r.ReponseApportee)
	}
	if r.PrixPropose != 0 {
		t.Errorf("expected zero PrixPropose, got %f", r.PrixPropose)
	}
	if r.Gagne {
		t.Error("expected Gagne to be false by default")
	}
}

func TestReponseHistoriqueFieldAssignment(t *testing.T) {
	tests := []struct {
		name  string
		input ReponseHistorique
	}{
		{
			name: "winning response",
			input: ReponseHistorique{
				ID:                "RH-001",
				AppelOffreID:      "AO-2023-BTP-LYON",
				ExigenceTechnique: "Pose de revêtements de sol souples en milieu hospitalier",
				ReponseApportee:   "Utilisation de PVC homogène soudé à chaud",
				PrixPropose:       125000.50,
				Gagne:             true,
			},
		},
		{
			name: "losing response",
			input: ReponseHistorique{
				ID:                "RH-002",
				AppelOffreID:      "AO-2023-BTP-LYON",
				ExigenceTechnique: "Installation électrique",
				ReponseApportee:   "Câblage standard",
				PrixPropose:       80000.00,
				Gagne:             false,
			},
		},
		{
			name: "zero price response",
			input: ReponseHistorique{
				ID:           "RH-003",
				AppelOffreID: "AO-GRATUIT",
				PrixPropose:  0,
				Gagne:        false,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.input
			if r.ID != tc.input.ID {
				t.Errorf("ID mismatch: got %q, want %q", r.ID, tc.input.ID)
			}
			if r.AppelOffreID != tc.input.AppelOffreID {
				t.Errorf("AppelOffreID mismatch: got %q, want %q", r.AppelOffreID, tc.input.AppelOffreID)
			}
			if r.ExigenceTechnique != tc.input.ExigenceTechnique {
				t.Errorf("ExigenceTechnique mismatch: got %q, want %q", r.ExigenceTechnique, tc.input.ExigenceTechnique)
			}
			if r.ReponseApportee != tc.input.ReponseApportee {
				t.Errorf("ReponseApportee mismatch: got %q, want %q", r.ReponseApportee, tc.input.ReponseApportee)
			}
			if r.PrixPropose != tc.input.PrixPropose {
				t.Errorf("PrixPropose mismatch: got %f, want %f", r.PrixPropose, tc.input.PrixPropose)
			}
			if r.Gagne != tc.input.Gagne {
				t.Errorf("Gagne mismatch: got %v, want %v", r.Gagne, tc.input.Gagne)
			}
		})
	}
}

func TestReponseHistoriqueNegativePrice(t *testing.T) {
	r := ReponseHistorique{
		ID:          "RH-NEG",
		PrixPropose: -500.0,
	}
	if r.PrixPropose != -500.0 {
		t.Errorf("expected PrixPropose -500.0, got %f", r.PrixPropose)
	}
}

func TestAppelOffreDateLimiteTimezone(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("Europe/Paris timezone not available:", err)
	}
	deadline := time.Date(2025, 3, 15, 12, 0, 0, 0, loc)
	ao := AppelOffre{
		ID:         "AO-TZ",
		DateLimite: deadline,
	}
	if ao.DateLimite.Location().String() != "Europe/Paris" {
		t.Errorf("expected Europe/Paris timezone, got %s", ao.DateLimite.Location().String())
	}
	if ao.DateLimite.UTC().Hour() != deadline.UTC().Hour() {
		t.Errorf("timezone conversion mismatch")
	}
}