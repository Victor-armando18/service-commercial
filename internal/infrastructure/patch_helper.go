package infrastructure

import (
	"encoding/json"
	"fmt"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	jsonpatch "github.com/evanphx/json-patch/v5"
)

// ApplyOrderPatch recebe o pedido original e os deltas, retornando o pedido atualizado.
func ApplyOrderPatch(original domain.Order, patchData []byte) (domain.Order, error) {
	// 1. Converter a struct original para JSON
	originalJSON, _ := json.Marshal(original)

	// 2. Aplicar o patch (RFC 6902)
	patch, err := jsonpatch.DecodePatch(patchData)
	if err != nil {
		return original, fmt.Errorf("falha ao decodificar patch: %w", err)
	}

	modifiedJSON, err := patch.Apply(originalJSON)
	if err != nil {
		return original, fmt.Errorf("falha ao aplicar patch: %w", err)
	}

	// 3. Converter de volta para a struct Order
	var updatedOrder domain.Order
	if err := json.Unmarshal(modifiedJSON, &updatedOrder); err != nil {
		return original, err
	}

	return updatedOrder, nil
}
