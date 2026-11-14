package dataset

import "testing"

func TestFullTrainingDataset(t *testing.T) {
	data, e := Load("../../../data/synthetic", "2026-10-28", "2026-10-28T08:30:00+05:00")
	if e != nil {
		t.Fatal(e)
	}
	if len(data.Reference.Staff) != 40 || len(data.Reference.ServiceCenters) != 5 || len(data.Day.Orders) != 400 || len(data.Matrix.Points) != 105 {
		t.Fatal("wrong full training dataset size")
	}
	ltp, install := 0, 0
	for _, s := range data.Reference.Staff {
		if s.Profiles[0] == "LTP3" {
			ltp++
		}
		if s.Profiles[0] == "INSTALL" {
			install++
		}
	}
	if ltp != 28 || install != 12 {
		t.Fatal("wrong staff profile distribution")
	}
	works := data.Reference.WorkTypes
	if works[2].MaxStartDelayMin == nil || *works[2].MaxStartDelayMin != 10 || works[0].MaxStartDelayMin != nil {
		t.Fatal("CSV nullable/10-minute installation rule lost")
	}
}
