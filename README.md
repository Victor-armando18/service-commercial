# Commercial Rules Engine & POS System

Motor de cálculo autoritativo baseado em **JsonLogic** com pipeline de execução por fases, projetado para sistemas comerciais que exigem determinismo e reconciliação entre Front-end e Back-end.

---

## 🚀 Arquitetura da Engine

A engine opera sob o conceito de **Pipeline de Fases**. Diferente de motores de regras lineares, esta implementação obriga a execução sequencial de 6 fases para garantir que os cálculos dependentes (ex: impostos sobre valores já descontados) sejam processados na ordem correta.

### 1. Fases de Execução
| Fase | Descrição | Objetivo |
| :--- | :--- | :--- |
| `baseline` | Recálculo Bruto | Ignora valores enviados pelo front e recalcula `baseValue` dos itens. |
| `orderAdjust` | Ajustes de Cabeçalho | Aplica descontos globais, acréscimos ou fretes. |
| `itemAdjust` | Ajustes de Itens | Regras específicas por SKU ou categoria (ex: Leve 3 Pague 2). |
| `taxes` | Cálculo de Impostos | Aplicação de VAT/IVA sobre o valor líquido recalculado. |
| `totals` | Fechamento | Consolida o `totalValue` final do objeto. |
| `guards` | Segurança | Validações de compliance (ex: bloqueio se total > limite). |

---

## 🛠 Estrutura do RulePack (JSON)

As regras são definidas de forma declarativa. Cada regra possui uma `phase` e um `output_key` que define onde o resultado da `logic` será gravado no estado.

```json
{
  "version": "v1.2",
  "rules": [
    {
      "id": "R_CALC_BASE",
      "phase": "baseline",
      "logic": {
        "round": [
          {
            "foreach": [
              { "var": "order.items" },
              { "*": [{ "var": "item.value" }, { "var": "item.qty" }] }
            ]
          },
          2
        ]
      },
      "output_key": "order.baseValue"
    }
  ]
}
``` 
## 📡 Integração e Reconciliação 
A Engine foi desenhada para resolver o problema de "preços divergentes" entre UI e Servidor através de:

* StateFragment: O servidor não retorna apenas "OK". Ele retorna o fragmento do objeto recalculado.
* ServerDelta: Diferença explícita entre o valor sugerido pelo Front e o valor imposto pela Engine.
* Determinismo: Uso de rulesVersion para garantir que o cálculo feito hoje seja idêntico ao de amanhã, mesmo que as regras globais mudem.

## 💻 Como Executar 
Pré-requisitos
Go 1.20+

Estrutura de pastas: data/rules e data/db

Iniciar o Servidor (PEP/Backend)
```bash
go run cmd/engine/main.go
```

Iniciar a Ferramenta de Diagnóstico (CLI)
A CLI permite inspecionar o stateFragment e os ExecutionLogs detalhadamente:

```bash
go run cmd/external-app/main.go
```

## 📂 Estrutura de Dados (Sales) 
As vendas persistidas seguem o formato consolidado pela Engine:

```json
[
  {
    "id": "SALE-20260124033914",
    "currency": "AOA",
    "baseValue": 2125,
    "items": [
      {
        "sku": "PROD-002",
        "value": 2500,
        "qty": 1
      }
    ],
    "appliedTaxes": {
      "VAT": 297.5
    },
    "totalItems": 1,
    "discountPercentage": 0.15,
    "totalValue": 2422.5,
    "rulesVersion": "v1.2",
    "correlationId": "CORR-SALE-20260124033914"
  }
]
``` 
## 🧪Diagnóstico e Logs 
A Engine produz logs detalhados por cada regra executada:

* Rule ID: Qual regra foi disparada.
* Me*ssage: Descrição da operação realizada.
* GuardsHit: Lista de violações de segurança que impediram a persistência.
