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
	if ao.RegelementConsultation != "" {
		t.Errorf("expected empty RegelementConsultation, got %q", ao.RegelementConsultation)
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

func TestAppelOffreInitialization(t *testing.T) {
	deadline := time.Date(2024, 6, 30, 17, 0, 0, 0, time.UTC)
	ao := AppelOffre{
		ID:                     "AO-2024-001",
		Titre:                  "Construction de bâtiments scolaires",
		MaitreDouvrage:         "Mairie de Paris",
		DateLimite:             deadline,
		RegelementConsultation: "RC-2024",
		CCTP:                   "Cahier des clauses techniques particulières",
		CCAP:                   "Cahier des clauses administratives particulières",
		BPU_DPGF:               "Bordereau de prix unitaires",
	}

	if ao.ID != "AO-2024-001" {
		t.Errorf("expected ID %q, got %q", "AO-2024-001", ao.ID)
	}
	if ao.Titre != "Construction de bâtiments scolaires" {
		t.Errorf("expected Titre %q, got %q", "Construction de bâtiments scolaires", ao.Titre)
	}
	if ao.MaitreDouvrage != "Mairie de Paris" {
		t.Errorf("expected MaitreDouvrage %q, got %q", "Mairie de Paris", ao.MaitreDouvrage)
	}
	if !ao.DateLimite.Equal(deadline) {
		t.Errorf("expected DateLimite %v, got %v", deadline, ao.DateLimite)
	}
	if ao.CCTP != "Cahier des clauses techniques particulières" {
		t.Errorf("expected CCTP %q, got %q", "Cahier des clauses techniques particulières", ao.CCTP)
	}
	if ao.CCAP != "Cahier des clauses administratives particulières" {
		t.Errorf("expected CCAP %q, got %q", "Cahier des clauses administratives particulières", ao.CCAP)
	}
	if ao.BPU_DPGF != "Bordereau de prix unitaires" {
		t.Errorf("expected BPU_DPGF %q, got %q", "Bordereau de prix unitaires", ao.BPU_DPGF)
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
	if r.PrixPropose != 0.0 {
		t.Errorf("expected PrixPropose 0.0, got %f", r.PrixPropose)
	}
	if r.Gagne != false {
		t.Errorf("expected Gagne false, got %v", r.Gagne)
	}
}

func TestReponseHistoriqueInitialization(t *testing.T) {
	r := ReponseHistorique{
		ID:                "rh-uuid-001",
		AppelOffreID:      "AO-2023-BTP-LYON",
		ExigenceTechnique: "Détailler la méthodologie pour la pose de revêtements de sol.",
		ReponseApportee:   "Nous utiliserons du PVC homogène soudé à chaud.",
		PrixPropose:       125000.50,
		Gagne:             true,
	}

	if r.ID != "rh-uuid-001" {
		t.Errorf("expected ID %q, got %q", "rh-uuid-001", r.ID)
	}
	if r.AppelOffreID != "AO-2023-BTP-LYON" {
		t.Errorf("expected AppelOffreID %q, got %q", "AO-2023-BTP-LYON", r.AppelOffreID)
	}
	if r.ExigenceTechnique != "Détailler la méthodologie pour la pose de revêtements de sol." {
		t.Errorf("unexpected ExigenceTechnique: %q", r.ExigenceTechnique)
	}
	if r.ReponseApportee != "Nous utiliserons du PVC homogène soudé à chaud." {
		t.Errorf("unexpected ReponseApportee: %q", r.ReponseApportee)
	}
	if r.PrixPropose != 125000.50 {
		t.Errorf("expected PrixPropose %f, got %f", 125000.50, r.PrixPropose)
	}
	if !r.Gagne {
		t.Errorf("expected Gagne true, got false")
	}
}

func TestReponseHistoriqueGagneFalse(t *testing.T) {
	r := ReponseHistorique{
		ID:           "rh-uuid-002",
		AppelOffreID: "AO-2024-INFRA",
		Gagne:        false,
	}
	if r.Gagne {
		t.Errorf("expected Gagne false, got true")
	}
}

func TestReponseHistoriquePrixZero(t *testing.T) {
	r := ReponseHistorique{
		ID:          "rh-uuid-003",
		PrixPropose: 0.0,
	}
	if r.PrixPropose != 0.0 {
		t.Errorf("expected PrixPropose 0.0, got %f", r.PrixPropose)
	}
}

func TestReponseHistoriqueNegativePrice(t *testing.T) {
	// PrixPropose is a float64 — no validation constraint in the struct,
	// so negative values are representable. This is a boundary case.
	r := ReponseHistorique{
		ID:          "rh-uuid-004",
		PrixPropose: -1.0,
	}
	if r.PrixPropose != -1.0 {
		t.Errorf("expected PrixPropose -1.0, got %f", r.PrixPropose)
	}
}

func TestAppelOffreDateLimiteIsTime(t *testing.T) {
	// Verify DateLimite field holds a time.Time value correctly.
	now := time.Now().UTC().Truncate(time.Second)
	ao := AppelOffre{DateLimite: now}
	if !ao.DateLimite.Equal(now) {
		t.Errorf("expected DateLimite %v, got %v", now, ao.DateLimite)
	}
}