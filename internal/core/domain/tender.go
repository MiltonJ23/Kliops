package domain


import "time"


// AppelOffre refers to the complete consulation file (DCE)
type AppelOffre struct {
	ID string 
	Titre string 
	MaitreDouvrage string 
	DateLimite time.Time 
	RegelementConsultation string 
	CCTP string // Specifications of Special Technical Clauses 
	CCAP string // Specifications of Special Administrative Clauses
	BPU_DPGF string // Price Schedule or Price Breakdown 
}

type ReponseHistorique struct {
	ID string 
	AppelOffreID string 
	ExigenceTechnique string // abstract of the CCTP related to the AppelOffre it was suppose to answer linked through AppelOffreID 
	ReponseApportee string // Technical Brief which was written to respond to the call for tenders
	PrixPropose float64 
	Gagne bool 
}