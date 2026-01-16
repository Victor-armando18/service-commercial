package diff

type Differ struct{}

func (d *Differ) Diff(before, after map[string]any) map[string]any {
	delta := map[string]any{}
	for k, v := range after {
		if before[k] != v {
			delta[k] = v
		}
	}
	return delta
}
