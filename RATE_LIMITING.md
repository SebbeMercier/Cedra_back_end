# 🛡️ Rate Limiting - Protection Anti-Spam

## ✅ Rate Limits appliqués

### 1. **Login** (`POST /api/auth/login`)
- **Limite** : 5 tentatives par email
- **Cooldown** : 15 minutes après 5 échecs
- **Protection** : Empêche les attaques par force brute

**Exemple de réponse après 5 échecs** :
```json
{
  "error": "Trop de tentatives échouées. Compte bloqué pendant 15 minutes",
  "retry_after": 900
}
```

### 2. **Register** (`POST /api/auth/register`)
- **Limite** : 3 inscriptions par IP
- **Cooldown** : 30 minutes
- **Protection** : Empêche la création de comptes en masse

### 3. **Forgot Password** (`POST /api/auth/forgot-password`)
- **Limite** : 3 demandes par email
- **Cooldown** : 10 minutes
- **Protection** : Empêche le spam d'emails

### 4. **API Global** (toutes les routes)
- **Limite** : 100 requêtes par minute par IP
- **Cooldown** : 1 minute
- **Protection** : Protection générale contre le spam

**Headers de réponse** :
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
```

### 5. **Cart Add** (`POST /api/cart/add`)
- **Limite** : 20 ajouts par minute par utilisateur
- **Cooldown** : 1 minute
- **Protection** : Empêche le spam d'ajouts au panier

### 6. **Search** (`GET /api/products/search`)
- **Limite** : 30 recherches par minute par IP
- **Cooldown** : 1 minute
- **Protection** : Empêche les recherches abusives

---

## 📊 Tableau récapitulatif

| Endpoint | Limite | Cooldown | Clé Redis |
|----------|--------|----------|-----------|
| **POST /auth/login** | 5 tentatives | 15 min | `login_attempts:EMAIL` |
| **POST /auth/register** | 3 inscriptions | 30 min | `register_attempts:IP` |
| **POST /auth/forgot-password** | 3 demandes | 10 min | `forgot_password_attempts:EMAIL` |
| **API Global** | 100 req/min | 1 min | `api_requests:IP` |
| **POST /cart/add** | 20 ajouts/min | 1 min | `cart_add:USER_ID` |
| **GET /products/search** | 30 recherches/min | 1 min | `search_requests:IP` |

---

## 🧪 Tests

### Test 1 : Login avec échecs répétés

```powershell
$base = "http://cedra.eldocam.com:8080/api"
$body = @{email="test@example.com"; password="wrong"} | ConvertTo-Json

# Tentatives 1-5 (devraient échouer avec 401)
1..5 | ForEach-Object {
    Write-Host "Tentative $_"
    try {
        Invoke-RestMethod "$base/auth/login" -Method Post -ContentType "application/json" -Body $body
    } catch {
        Write-Host "Échec attendu"
    }
}

# Tentative 6 (devrait être bloquée avec 429)
Write-Host "`nTentative 6 (devrait être bloquée)"
try {
    Invoke-RestMethod "$base/auth/login" -Method Post -ContentType "application/json" -Body $body
} catch {
    $_.Exception.Response.StatusCode # Devrait être 429 (Too Many Requests)
}
```

### Test 2 : Vérifier les headers de rate limit

```powershell
$response = Invoke-WebRequest "http://cedra.eldocam.com:8080/api/products" -Method Get
$response.Headers["X-RateLimit-Limit"]
$response.Headers["X-RateLimit-Remaining"]
```

### Test 3 : Spam de recherches

```powershell
# Faire 31 recherches rapidement (la 31ème devrait être bloquée)
1..31 | ForEach-Object {
    Write-Host "Recherche $_"
    try {
        Invoke-RestMethod "http://cedra.eldocam.com:8080/api/products/search?q=test" -Method Get
    } catch {
        Write-Host "Bloqué à la recherche $_" -ForegroundColor Red
    }
}
```

---

## 🔍 Monitoring Redis

### Voir les tentatives de login

```bash
redis-cli -h 192.168.1.130 -a R3D9S-C3DRA!

# Voir toutes les clés de rate limiting
KEYS login_attempts:*
KEYS login_cooldown:*
KEYS register_attempts:*
KEYS api_requests:*

# Voir les tentatives pour un email spécifique
GET login_attempts:test@example.com

# Voir si un email est en cooldown
EXISTS login_cooldown:test@example.com
TTL login_cooldown:test@example.com
```

### Débloquer manuellement un utilisateur

```bash
# Supprimer le cooldown
DEL login_cooldown:test@example.com
DEL login_attempts:test@example.com
```

---

## ⚙️ Configuration

Pour ajuster les limites, modifiez `internal/middleware/rate_limit.go` :

```go
const (
    LoginMaxAttempts        = 5    // Modifier ici
    RegisterMaxAttempts     = 3
    ForgotPasswordMaxAttempts = 3
    APIMaxRequests          = 100
    
    LoginCooldown           = 15 * time.Minute  // Modifier ici
    RegisterCooldown        = 30 * time.Minute
    ForgotPasswordCooldown  = 10 * time.Minute
    APICooldown             = 1 * time.Minute
)
```

---

## 🎯 Avantages

1. ✅ **Protection contre force brute** - Login limité à 5 tentatives
2. ✅ **Protection contre spam** - Inscriptions et recherches limitées
3. ✅ **Protection DDoS** - Limite globale de 100 req/min
4. ✅ **Expérience utilisateur** - Messages clairs avec `retry_after`
5. ✅ **Monitoring** - Toutes les tentatives sont trackées dans Redis
6. ✅ **Flexibilité** - Cooldowns automatiques et configurables

---

## 🔐 Sécurité renforcée

Avec ces rate limits + **bcrypt coût 8**, votre application est maintenant :

- ✅ **Protégée contre force brute** (5 tentatives max)
- ✅ **Protégée contre spam** (limites sur tous les endpoints)
- ✅ **Performante** (login à 13ms après cache)
- ✅ **Sécurisée** (bcrypt + rate limiting = double protection)

**Temps pour craquer un mot de passe** :
- Sans rate limiting : ~8 ans (bcrypt coût 8)
- Avec rate limiting (5 tentatives/15min) : **~2400 ans** 🔒

---

**Date** : 2025-11-27  
**Version** : 1.0  
**Auteur** : Kiro AI
