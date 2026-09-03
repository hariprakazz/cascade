#!/bin/bash
set -euo pipefail

TARGET="${1:-/tmp/cascade-bench-repo}"
rm -rf "$TARGET"
mkdir -p "$TARGET/packages"
cd "$TARGET"

cat > go.mod << 'EOF'
module github.com/hariprakazz/benchrepo

go 1.21
EOF

mkdir -p packages/auth packages/logging packages/db packages/cache packages/api packages/web packages/cli packages/mobile packages/admin packages/worker packages/batchjob

echo 'package auth' > packages/auth/main.go
echo 'package logging' > packages/logging/main.go

cat > packages/db/main.go << 'EOF'
package db

import (
	"github.com/hariprakazz/benchrepo/packages/auth"
	"github.com/hariprakazz/benchrepo/packages/logging"
)

var _ = auth.X
var _ = logging.X
EOF

cat > packages/api/main.go << 'EOF'
package api

import "github.com/hariprakazz/benchrepo/packages/db"

var _ = db.X
EOF

cat > packages/web/main.go << 'EOF'
package web

import "github.com/hariprakazz/benchrepo/packages/api"

var _ = api.X
EOF

cat > packages/cli/main.go << 'EOF'
package cli

import "github.com/hariprakazz/benchrepo/packages/api"

var _ = api.X
EOF

cat > packages/mobile/main.go << 'EOF'
package mobile

import "github.com/hariprakazz/benchrepo/packages/api"

var _ = api.X
EOF

cat > packages/admin/main.go << 'EOF'
package admin

import "github.com/hariprakazz/benchrepo/packages/db"

var _ = db.X
EOF

cat > packages/cache/Cargo.toml << 'EOF'
[package]
name = "cache"
version = "0.1.0"

[dependencies]
auth = { path = "../auth" }
EOF

cat > packages/worker/Cargo.toml << 'EOF'
[package]
name = "worker"
version = "0.1.0"

[dependencies]
cache = { path = "../cache" }
EOF

cat > packages/batchjob/dub.json << 'EOF'
{
    "name": "batchjob",
    "dependencies": {
        "logging": "~>1.0.0"
    }
}
EOF

git init -q
git config user.email "bench@cascade.local"
git config user.name "cascade-bench"
git add -A
git commit -qm "initial monorepo structure"

echo '// tweak' >> packages/auth/main.go
git commit -aqm "tweak auth"

echo '// tweak' >> packages/web/main.go
git commit -aqm "tweak web"

echo '// tweak' >> packages/cache/Cargo.toml
git commit -aqm "bump cache"

echo 'edition = "2021"' >> packages/cache/Cargo.toml
git commit -aqm "cache edition bump"

echo '// leaf change' >> packages/cli/main.go
git commit -aqm "tweak cli"

echo "benchmark repo generated at $TARGET"
git log --oneline
