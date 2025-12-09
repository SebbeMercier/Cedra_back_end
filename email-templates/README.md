# 📧 Email Templates - React Email

Templates email créés avec React et compilés en HTML pour le backend Go.

## 🚀 Workflow

### 1. Créer/Modifier un template React

Les templates sont dans `emails/`:
- `welcome.jsx` - Email de bienvenue
- `order-confirmation.jsx` - Confirmation de commande

### 2. Compiler les templates

```bash
npm run build
```

Cette commande:
- Compile les templates React en HTML
- Génère les fichiers dans `../internal/templates/`
- Les templates utilisent des placeholders Go: `{{.Variable}}`

### 3. Utiliser dans Go

```go
import "cedra_back_end/internal/utils"

// Email de bienvenue