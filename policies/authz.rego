package dolphin.authz

import rego.v1

# Default deny: segurança em primeiro lugar
default allow := false

# Action Specs: Define os limites (constraints) por permissão
action_specs := {
    "order.discount.apply": {
        "kind": "max",
        "constraint_key": "max_discount_pct",
        "attribute_key": "max_allowed_discount",
        "default_max": 5, # 5% por padrão se o usuário não tiver atributo
        "unlimited_permission": "sales.admin"
    }
}

# Regra principal de permissão (RBAC)
allow if {
    input.subject.permissions[_] == input.action
}

# Geração de Capabilities e Constraints para o Front-end
capabilities := { action: info |
    some action, spec in action_specs
    info := {
        "allowed": input.subject.permissions[_] == action,
        "constraints": { spec.constraint_key: get_limit(action) }
    }
}

# Helper para calcular o limite (Constraint) baseado nos atributos do usuário
get_limit(action) := limit if {
    spec := action_specs[action]
    # Se for admin, o limite é 100%
    input.subject.permissions[_] == spec.unlimited_permission
    limit := 100
} else := limit if {
    spec := action_specs[action]
    limit := input.subject.attributes[spec.attribute_key]
} else := limit if {
    spec := action_specs[action]
    limit := spec.default_max
}