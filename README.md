# Redis Clone

Un clone simple de Redis implémenté en Go, supportant les principales commandes et le protocole RESP.

## 🚀 Fonctionnalités

### Commandes supportées

#### Commandes de base
- `PING` - Test de connectivité
- `SET key value` - Définir une valeur string
- `GET key` - Récupérer une valeur string
- `DEL key [key ...]` - Supprimer une ou plusieurs clés
- `EXISTS key [key ...]` - Vérifier l'existence de clés

#### Commandes numériques
- `INCR key` - Incrémenter une valeur numérique
- `DECR key` - Décrémenter une valeur numérique

#### Gestion des expirations
- `EXPIRE key seconds` - Définir une expiration
- `TTL key` - Obtenir le temps de vie restant

#### Commandes Hash
- `HSET key field value` - Définir un champ dans un hash
- `HGET key field` - Récupérer un champ d'un hash
- `HDEL key field [field ...]` - Supprimer des champs d'un hash

#### Commandes utilitaires
- `KEYS pattern` - Lister les clés (pattern "*" supporté)
- `INFO` - Informations sur la base de données
- `DBSIZE` - Nombre de clés dans la base
- `TYPE key` - Type d'une clé

### Fonctionnalités avancées
- ✅ **Protocole RESP** complet
- ✅ **Multi-threading** avec verrous RWMutex
- ✅ **Gestion des expirations** automatique
- ✅ **Persistance AOF/RDB** (optionnelle)
- ✅ **CLI compatible** avec les commandes Redis standard
- ✅ **Types de données** : String, Hash
- ✅ **Graceful shutdown**

## 📁 Structure du projet

```
redis-clone/
├── cmd/
│   ├── server/           # Point d'entrée du serveur
│   │   └── main.go
│   └── cli/              # Client en ligne de commande
│       └── main.go
├── internal/
│   ├── server/           # Logique du serveur
│   │   ├── server.go     # Serveur principal
│   │   ├── client.go     # Gestion des clients
│   │   └── commands.go   # Implémentation des commandes
│   ├── database/         # Moteur de base de données
│   │   └── database.go
│   ├── protocol/         # Protocole RESP
│   │   └── resp.go
│   └── persistence/      # Persistance AOF/RDB
│       └── persistence.go
├── Makefile
├── go.mod
└── README.md
```

## 🛠️ Installation et utilisation

### Prérequis
- Go 1.19+ installé
- Git

### Installation
```bash
git clone https://github.com/votre-username/redis-clone.git
cd redis-clone
go mod tidy
```

### Démarrage du serveur
```bash
# Méthode 1: Avec Makefile
make run

# Méthode 2: Directement avec Go
go run cmd/server/main.go -port 6379
```

Le serveur démarre par défaut sur le port 6379.

### Utilisation du CLI
```bash
# Dans un autre terminal
make run-cli

# Ou directement
go run cmd/cli/main.go localhost:6379
```

## 📝 Exemples d'utilisation

### Session basique
```bash
redis> PING
PONG

redis> SET user:1 "Alice"
OK

redis> GET user:1
"Alice"

redis> INCR counter
(integer) 1

redis> EXPIRE counter 60
(integer) 1

redis> TTL counter
(integer) 59
```

### Commandes Hash
```bash
redis> HSET user:profile name "Bob"
(integer) 1

redis> HSET user:profile age "30"
(integer) 1

redis> HGET user:profile name
"Bob"

redis> HDEL user:profile age
(integer) 1
```

### Inspection de la base
```bash
redis> KEYS *
1) "user:1"
2) "counter"
3) "user:profile"

redis> TYPE user:1
string

redis> TYPE user:profile
hash
```

## ⚙️ Configuration

### Options du serveur
```bash
go run cmd/server/main.go -port 6379 -config redis.conf
```

### Fichiers de configuration
- `redis.conf` - Configuration principale (optionnel)
- `appendonly.aof` - Journal des commandes (AOF)
- `dump.rdb` - Sauvegarde binaire (RDB)

## 🏗️ Architecture

### Base de données en mémoire
- **Stockage** : `map[string]*Value` pour les données principales
- **Expirations** : `map[string]time.Time` pour les TTL
- **Concurrence** : `sync.RWMutex` pour les accès thread-safe
- **Types** : Support String et Hash, extensible pour List/Set

### Protocole RESP
- **Parsing** complet du protocole Redis (REdis Serialization Protocol)
- **Types supportés** : Simple String, Error, Integer, Bulk String, Array
- **Compatibilité** avec les clients Redis existants

### Persistance
- **AOF** : Append-Only File pour rejouer les commandes
- **RDB** : Snapshots binaires périodiques
- **Background saving** : Sauvegarde automatique configurable

## 🧪 Tests

### Tests manuels
Utilisez le CLI fourni pour tester toutes les commandes :

```bash
# Démarrer le serveur
make run-server

# Dans un autre terminal
make run-cli

# Exécuter les tests
redis> SET test "Hello World"
redis> GET test
redis> INCR counter
redis> KEYS *
```

## 📊 Performance

### Caractéristiques
- **Opérations** : ~50,000 SET/GET par seconde (dépend du hardware)
- **Mémoire** : Environ 50 bytes par clé string
- **Concurrence** : Support multiple clients simultanés
- **Latence** : < 1ms pour les opérations simples

### Limitations
- **Mémoire limitée** : Toutes les données en RAM
- **Pas de clustering** : Instance unique seulement
- **Types limités** : String et Hash uniquement
- **Pas de réplication** : Pas de master/slave

## 🤝 Contribution

1. Fork le projet
2. Créer une branche feature (`git checkout -b feature/nouvelle-commande`)
3. Commit les changements (`git commit -am 'Ajout nouvelle commande'`)
4. Push vers la branche (`git push origin feature/nouvelle-commande`)
5. Créer une Pull Request
