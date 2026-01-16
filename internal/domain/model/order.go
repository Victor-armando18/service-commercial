package model

type Order struct {
	ID    string
	Items []Item
}

type Item struct {
	SKU      string
	Quantity int
	Price    float64
}

func (o Order) ToMap() map[string]any {
	items := make([]any, len(o.Items))
	for i, item := range o.Items {
		items[i] = map[string]any{
			"SKU":      item.SKU,
			"Quantity": item.Quantity,
			"Price":    item.Price,
		}
	}
	return map[string]any{
		"ID":    o.ID,
		"Items": items,
	}
}
