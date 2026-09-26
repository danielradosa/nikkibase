package pipeline

func MergeEntries(preferred, fallback []Entry) []Entry {
	at := make(map[int]int, len(preferred))
	out := make([]Entry, len(preferred))
	copy(out, preferred)
	for i, e := range out {
		at[e.Item.ID] = i
	}
	for _, e := range fallback {
		i, seen := at[e.Item.ID]
		if !seen {
			at[e.Item.ID] = len(out)
			out = append(out, e)
			continue
		}
		if out[i].Rarity == 0 {
			out[i].Rarity = e.Rarity
		}
		if out[i].Suit == "" {
			out[i].Suit = e.Suit
		}
		if len(out[i].Item.Tags) == 0 {
			out[i].Item.Tags = e.Item.Tags
		}
		if out[i].Item.FlatBonus == 0 {
			out[i].Item.FlatBonus = e.Item.FlatBonus
		}
		for p := range 5 {
			if out[i].Grades[p] == "" && e.Grades[p] != "" {
				out[i].Item.Attrs[p] = e.Item.Attrs[p]
				out[i].Item.Stats[p] = Stat(e.Grades[p], out[i].Item.Slot)
				out[i].Grades[p] = e.Grades[p]
			}
		}
	}
	return out
}
