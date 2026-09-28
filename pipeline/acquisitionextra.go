package pipeline

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type ExtraAcquisition struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Text  string `json:"text"`
	Basis string `json:"basis"`
}

func ReadAcquisitionExtra(raw []byte) ([]ExtraAcquisition, error) {
	var doc struct {
		Items []ExtraAcquisition `json:"items"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("acquisition extra: %w", err)
	}
	seen := map[int]bool{}
	for _, e := range doc.Items {
		switch {
		case seen[e.ID]:
			return nil, fmt.Errorf("acquisition extra: %d is listed twice", e.ID)
		case !slices.Contains(AcquisitionKinds, e.Kind):
			return nil, fmt.Errorf("acquisition extra: %d has kind %q, which is none of %v", e.ID, e.Kind, AcquisitionKinds)
		case strings.TrimSpace(e.Text) == "" || strings.TrimSpace(e.Basis) == "" || strings.TrimSpace(e.Name) == "":
			return nil, fmt.Errorf("acquisition extra: %d must give its name, a line and the basis for it", e.ID)
		}
		seen[e.ID] = true
	}
	return doc.Items, nil
}

func ApplyAcquisitionExtra(acq map[int][]Acquisition, cat AcquisitionCatalogue, extra []ExtraAcquisition) error {
	var wrong []string
	for _, e := range extra {
		name, ok := cat.Names[e.ID]
		switch {
		case !ok:
			wrong = append(wrong, fmt.Sprintf("%d is not in the catalogue", e.ID))
		case !sameGarment(name, e.Name):
			wrong = append(wrong, fmt.Sprintf("%d is %q, not %q", e.ID, name, e.Name))
		case len(acq[e.ID]) > 0:
			wrong = append(wrong, fmt.Sprintf("%d %s now has a line from the sources", e.ID, e.Name))
		default:
			acq[e.ID] = []Acquisition{{Kind: e.Kind, Text: e.Text, Past: pastSource(e.Kind, e.Text)}}
		}
	}
	if len(wrong) > 0 {
		return fmt.Errorf("acquisition extra: %d entries no longer fit what the sources give: %s",
			len(wrong), strings.Join(wrong, "; "))
	}
	return nil
}
